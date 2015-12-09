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

type work struct {
	endpoint string
	format   string
}

func worker(queue chan work, wg *sync.WaitGroup) {
	defer wg.Done()
	client := oaimi.NewCachingClient(ioutil.Discard)
	client.CacheDir = CacheDir
	for w := range queue {
		req := oaimi.Request{Verb: "ListRecords", Endpoint: w.endpoint, Prefix: w.format}
		err := client.Do(req)
		if err != nil {
			if Verbose {
				log.Printf("failed %s: %s", w.endpoint, err)
			}
			continue
		}
		if Verbose {
			log.Printf("done: %s", w.endpoint)
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

	queue := make(chan work)
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
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) > 0 {
			if !strings.HasPrefix(fields[0], "http") {
				fields[0] = "http://" + fields[0]
			}
		}
		switch len(fields) {
		case 0:
			continue
		case 1:
			queue <- work{endpoint: fields[0], format: "oai_dc"}
		default:
			queue <- work{endpoint: fields[0], format: fields[1]}
		}
	}

	close(queue)
	wg.Wait()
}
