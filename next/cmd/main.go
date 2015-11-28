package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/miku/oaimi/next"
)

func main() {
	client := next.NewBatchingClient()
	// client := next.NewClient()
	req := next.Request{
		Endpoint: "http://www.ssoar.info/OAIHandler/request",
		Verb:     "ListRecords",
		Prefix:   "oai_dc",
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	b, err := json.Marshal(resp)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(b))
}
