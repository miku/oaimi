package oaimi

import (
	"encoding/json"
	"fmt"
	"time"
)

// RepositoryInfo holds some information about the repository.
type RepositoryInfo struct {
	Endpoint string              `json:"endpoint,omitempty"`
	Elapsed  float64             `json:"elapsed,omitempty"`
	About    Identify            `json:"about,omitempty"`
	Formats  ListMetadataFormats `json:"formats,omitempty"`
	Sets     ListSets            `json:"sets,omitempty"`
	Errors   []error             `json:"errors,omitempty"`
}

// MarshalJSON formats the RepositoryInfo a bit terser than the default
// serialization.
func (ri RepositoryInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"endpoint": ri.Endpoint,
		"elapsed":  ri.Elapsed,
		"id":       ri.About,
		"formats":  ri.Formats.Formats,
		"sets":     ri.Sets.Sets,
		"errors":   ri.Errors,
	})
}

// message is used to move around data from the request execution to the
// request collection.
type message struct {
	request  Request
	response Response
	err      error
}

var client = NewBatchingClient()

// doRequest executes a given OAI request and sends a message back a message.
// The request can be cancelled through the quit channel.
func doRequest(req Request, resp chan message, quit chan bool) {
	ch := make(chan message)
	go func() {
		r, err := client.Do(req)
		ch <- message{request: req, response: r, err: err}
	}()
	for {
		select {
		case <-quit:
			break
		case msg := <-ch:
			resp <- msg
			break
		}
	}
}

// AboutEndpoint returns information about a repository. Returns after at most
// 30 seconds.
func AboutEndpoint(endpoint string, timeout time.Duration) (*RepositoryInfo, error) {
	start := time.Now()

	resp := make(chan message)
	quit := make(chan bool)

	for _, verb := range []string{"Identify", "ListSets", "ListMetadataFormats"} {
		go doRequest(Request{Endpoint: endpoint, Verb: verb}, resp, quit)
	}

	info := &RepositoryInfo{Endpoint: endpoint, Errors: make([]error, 0)}
	defer func() {
		info.Elapsed = time.Since(start).Seconds()
	}()

	var received int

	expired := time.After(timeout)

	for {
		select {
		case msg := <-resp:
			if msg.err == nil {
				switch msg.request.Verb {
				case "Identify":
					info.About = msg.response.Identify
				case "ListSets":
					info.Sets = msg.response.ListSets
				case "ListMetadataFormats":
					info.Formats = msg.response.ListMetadataFormats
				}
			} else {
				info.Errors = append(info.Errors, msg.err)
			}
			received++
			if received == 3 {
				return info, nil
			}
		case <-expired:
			for i := 0; i < 3-received; i++ {
				quit <- true
			}
			return info, fmt.Errorf("timed out")
		}
	}
	return info, nil
}
