package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miku/oaimi"
)

var Verbose bool

func worker(queue, out chan string, timeout time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()
	for endpoint := range queue {
		ri, err := oaimi.AboutEndpoint(endpoint, timeout)
		if err != nil {
			if Verbose {
				log.Printf("failed %s: %s", endpoint, err)
			}
			continue
		}
		b, err := json.Marshal(ri)
		if err != nil {
			log.Fatal(err)
		}
		out <- string(b)
		if Verbose {
			log.Printf("done: %s", endpoint)
		}
	}
}

func writer(in chan string, done chan bool) {
	for s := range in {
		fmt.Println(s)
	}
	done <- true
}

func main() {
	timeout := flag.Duration("timeout", 30*time.Minute, "deadline for requests")
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
	out := make(chan string)
	done := make(chan bool)

	var wg sync.WaitGroup

	go writer(out, done)

	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(queue, out, *timeout, &wg)
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
	close(out)
	<-done
}
