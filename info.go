package oaimi

import (
	"fmt"
	"time"
)

// RepositoryInfo returns information about a repository. Returns after at
// most ten seconds.
func RepositoryInfo(endpoint string) (map[string]interface{}, error) {
	start := time.Now()
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
	var errors []error
	timeout := time.After(10 * time.Second)

	for {
		select {
		case msg := <-ch:
			if msg.err == nil {
				result[msg.key] = msg.value
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
		case <-timeout:
			return result, fmt.Errorf("timed out")
		}
	}
	return result, nil
}
