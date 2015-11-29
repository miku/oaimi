package next

import (
	"log"
	"time"
)

// RepositoryInfo returns information about a repository. Returns after at
// most ten seconds.
func RepositoryInfo(endpoint string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	client := NewBatchingClient()

	type message struct {
		key   string
		value interface{}
		err   error
	}

	ch := make(chan message)

	go func() {
		resp, err := client.Do(Request{Endpoint: endpoint, Verb: "Identify"})
		ch <- message{key: "id", value: resp.Identify, err: err}
	}()

	go func() {
		resp, err := client.Do(Request{Endpoint: endpoint, Verb: "ListSets"})
		ch <- message{key: "sets", value: resp.ListSets.Sets, err: err}
	}()

	go func() {
		resp, err := client.Do(Request{Endpoint: endpoint, Verb: "ListMetadataFormats"})
		ch <- message{key: "formats", value: resp.ListMetadataFormats.Formats, err: err}
	}()

	var received int
	timeout := time.After(10 * time.Second)

Loop:
	for {
		select {
		case msg := <-ch:
			if msg.err == nil {
				result[msg.key] = msg.value
			}
			received++
			if received == 3 {
				break Loop
			}
		case <-timeout:
			log.Println("operation timed out")
			break Loop
		}
	}
	return result, nil
}
