package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/miku/oaimi"
)

var Verbose bool

func worker(queue chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	client := oaimi.NewCachingClient(ioutil.Discard)
	for endpoint := range queue {
		req := oaimi.Request{Verb: "ListRecords", Endpoint: endpoint}
		err := client.Do(req)
		if err != nil {
			if Verbose {
				log.Printf("failed %s: %s", endpoint, err)
			}
			continue
		}
		if Verbose {
			log.Printf("done: %s", endpoint)
		}
	}
}

func main() {
	workers := flag.Int("w", 8, "requests in parallel")
	verbose := flag.Bool("verbose", false, "be verbose")
	showVersion := flag.Bool("v", false, "prints current program version")

	flag.Parse()

	if *showVersion {
		fmt.Println(oaimi.Version)
		os.Exit(0)
	}

	Verbose = *verbose
	oaimi.Verbose = *verbose

	var reader io.Reader
	var err error

	if flag.NArg() == 0 {
		reader = os.Stdin
	} else {
		reader, err = os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
	}

	queue := make(chan string)
	var wg sync.WaitGroup

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(queue, &wg)
	}

	rdr := bufio.NewReader(reader)
	for {
		line, err := rdr.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		endpoint := strings.TrimSpace(line)
		if endpoint == "" {
			continue
		}
		queue <- endpoint
	}

	close(queue)
	wg.Wait()
}
