package oaimi

import (
	"fmt"
	"log"
	"time"
)

type message struct {
	req  Request
	resp Response
	err  error
}

func doRequest(req Request, resp chan message, quit chan bool) {
	ch := make(chan message)
	go func() {
		client := NewBatchingClient()
		r, err := client.Do(req)
		ch <- message{req, r, err}
	}()
	for {
		select {
		case <-quit:
			log.Printf("stopping: %s on %s", req.Verb, req.Endpoint)
			break
		case msg := <-ch:
			resp <- msg
			break
		}
	}
}

// RepositoryInfo returns information about a repository. Returns after at
// most 30 seconds.
func RepositoryInfo(endpoint string) (map[string]interface{}, error) {
	start := time.Now()

	resp := make(chan message)
	quit := make(chan bool)

	go doRequest(Request{Endpoint: endpoint, Verb: "Identify"}, resp, quit)
	go doRequest(Request{Endpoint: endpoint, Verb: "ListSets"}, resp, quit)
	go doRequest(Request{Endpoint: endpoint, Verb: "ListMetadataFormats"}, resp, quit)

	result := make(map[string]interface{})
	var errors []error
	var received int

	for {
		select {
		case msg := <-resp:
			if msg.err == nil {
				switch msg.req.Verb {
				case "Identify":
					result["id"] = msg.resp.Identify
				case "ListSets":
					result["sets"] = msg.resp.ListSets.Sets
				case "ListMetadataFormats":
					result["formats"] = msg.resp.ListMetadataFormats.Formats
				}
			} else {
				errors = append(errors, msg.err)
			}
			received++
			if received == 3 {
				if len(errors) > 0 {
					result["errors"] = errors
				}
				result["elapsed"] = time.Since(start).Seconds()
				return result, nil
			}
		case <-time.After(120 * time.Second):
			for i := 0; i < 3-received; i++ {
				quit <- true
			}
			return result, fmt.Errorf("timed out")
		}
	}
	return result, nil
}
