package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/miku/oaimi"
	"github.com/mitchellh/go-homedir"
)

var Verbose bool
var CacheDir string

func worker(queue chan string, wg *sync.WaitGroup) {
	defer wg.Done()
	client := oaimi.NewCachingClient(ioutil.Discard)
	client.CacheDir = CacheDir
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

	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}

	workers := flag.Int("w", 8, "requests in parallel")
	verbose := flag.Bool("verbose", false, "be verbose")
	cacheDir := flag.String("cache", filepath.Join(home, oaimi.DefaultCacheDir), "where to cache responses")
	showVersion := flag.Bool("v", false, "prints current program version")

	flag.Parse()

	if *showVersion {
		fmt.Println(oaimi.Version)
		os.Exit(0)
	}

	CacheDir = *cacheDir
	Verbose = *verbose
	oaimi.Verbose = *verbose

	var reader io.Reader

	if flag.NArg() == 0 {
		reader = os.Stdin
	} else {
		var err error
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
