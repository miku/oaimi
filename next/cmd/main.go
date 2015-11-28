package main

import (
	"log"
	"os"

	"github.com/miku/oaimi/next"
)

func main() {
	// client := next.NewBatchingClient()
	// client := next.NewClient()

	// client := next.NewWriterClient(os.Stdout)
	// client.RootTag = "collection"

	client := next.NewCachingClient(os.Stdout)

	req := next.Request{
		Endpoint: "http://www.librelloph.com/oai",
		// Endpoint: "http://www.ssoar.info/OAIHandler/request",
		// Endpoint: "http://journals.sub.uni-hamburg.de/giga/afsp/oai",
		// Endpoint: "http://www.doabooks.org/oai",
		Verb: "ListRecords",
		// Prefix:   "oai_dc",
		Prefix: "marcxml",
	}
	err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	// b, err := json.Marshal(resp)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(string(b))
}
