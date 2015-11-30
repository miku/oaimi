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

	"github.com/miku/oaimi"
)

func worker(queue, out chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	for endpoint := range queue {
		m, err := oaimi.RepositoryInfo(endpoint)
		if err != nil {
			log.Printf("failed: %s", endpoint)
			continue
		}
		b, err := json.Marshal(m)
		if err != nil {
			log.Fatal(err)
		}
		out <- string(b)
		log.Printf("done: %s", endpoint)
	}
}

func writer(in chan string, done chan bool) {
	for s := range in {
		fmt.Println(s)
	}
	done <- true
}

func main() {
	workers := flag.Int("w", 8, "requests in parallel")
	verbose := flag.Bool("verbose", false, "be verbose")

	flag.Parse()

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
		go worker(queue, out, &wg)
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
