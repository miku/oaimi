package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/miku/oaimi/next"
)

func main() {

	showRepoInfo := flag.Bool("id", false, "show repository info")

	flag.Parse()

	if flag.NArg() == 0 {
		log.Fatal("endpoint URL required")
	}

	if *showRepoInfo {
		info, err := next.RepositoryInfo(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		b, err := json.Marshal(info)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(b))
		os.Exit(0)
	}
	// client := next.NewBatchingClient()
	// client := next.NewClient()

	// client := next.NewWriterClient(os.Stdout)
	// client.RootTag = "collection"

	client := next.NewCachingClient(os.Stdout)
	client.RootTag = "collection"

	req := next.Request{
		// Endpoint: "http://export.arxiv.org/oai2",
		// Endpoint: "http://www.librelloph.com/oai",
		Endpoint: "http://www.ssoar.info/OAIHandler/request",
		// Endpoint: "http://journals.sub.uni-hamburg.de/giga/afsp/oai",
		// Endpoint: "http://www.doabooks.org/oai",
		Verb: "ListRecords",
		// Verb: "ListSets",
		Prefix: "oai_dc",
		// Prefix: "marcxml",
		From: time.Date(2014, 11, 1, 0, 0, 0, 0, time.UTC),
		// Until: time.Date(2015, 11, 10, 0, 0, 0, 0, time.UTC),
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
