package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"time"

	"github.com/miku/oaimi"
)

func main() {
	var err error

	var defaultDir string
	usr, err := user.Current()
	if err == nil {
		defaultDir = path.Join(usr.HomeDir, ".oaimi")
	}

	cacheDir := flag.String("cache", defaultDir, "oaimi cache dir")
	set := flag.String("set", "", "OAI set")
	prefix := flag.String("prefix", "oai_dc", "OAI metadataPrefix")
	from := flag.String("from", "", "OAI from")
	until := flag.String("until", time.Now().Format("2006-01-02"), "OAI until")
	retry := flag.Int("retry", 10, "retry count for exponential backoff")
	dirname := flag.Bool("dirname", false, "show shard directory for request")
	verbose := flag.Bool("verbose", false, "more output")
	root := flag.String("root", "", "name of artificial root element tag to use")
	identify := flag.Bool("id", false, "show repository information")
	fromEarliest := flag.Bool("from-earliest", false, "harvest from earliest timestamp")
	showVersion := flag.Bool("v", false, "prints current program version")

	flag.Parse()

	if *showVersion {
		fmt.Println(oaimi.Version)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		log.Fatal("URL to OAI endpoint required")
	}

	endpoint := flag.Arg(0)

	if _, err := url.Parse(endpoint); err != nil {
		log.Fatal("endpoint is not an URL")
	}

	if *identify {
		var req oaimi.Request
		var err error

		req = oaimi.Request{Endpoint: endpoint, Verb: "Identify", Verbose: *verbose, MaxRetry: *retry}
		responseIdentify, err := req.DoOne()
		if err != nil {
			log.Fatal(err)
		}

		req = oaimi.Request{Endpoint: endpoint, Verb: "ListMetadataFormats", Verbose: *verbose, MaxRetry: *retry}
		responseFormats, err := req.DoOne()
		if err != nil {
			log.Fatal(err)
		}

		req = oaimi.Request{Endpoint: endpoint, Verb: "ListSets", Verbose: *verbose, MaxRetry: *retry}
		responseSets, err := req.DoOne()
		if err != nil {
			log.Fatal(err)
		}

		b, err := json.Marshal(map[string]interface{}{
			"identify": responseIdentify.Identify,
			"formats":  responseFormats.ListMetadataFormats.Formats,
			"sets":     responseSets.ListSets.Sets,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(b))
		os.Exit(0)
	}

	if *cacheDir == "" {
		log.Fatal("cache directory not set")
	}

	if *dirname {
		req := oaimi.CachedRequest{
			Cache: oaimi.Cache{Directory: *cacheDir},
			Request: oaimi.Request{
				Set:      *set,
				Prefix:   *prefix,
				Endpoint: endpoint,
			},
		}
		fmt.Println(filepath.Dir(req.Path()))
		os.Exit(0)
	}

	var From, Until time.Time

	if *from == "" || *fromEarliest {
		req := oaimi.Request{Endpoint: endpoint, Verb: "Identify", Verbose: *verbose, MaxRetry: *retry}
		resp, err := req.DoOne()
		if err != nil {
			log.Fatal(err)
		}
		if len(resp.Identify.EarliestDatestamp) < 10 {
			log.Fatalf("datestamp broken: %s - fix: use explicit -from parameter", resp.Identify.EarliestDatestamp)
		}
		if From, err = time.Parse("2006-01-02", resp.Identify.EarliestDatestamp[:10]); err != nil {
			log.Fatal(err)
		}
	} else {
		if From, err = time.Parse("2006-01-02", *from); err != nil {
			log.Fatal(err)
		}
	}

	if Until, err = time.Parse("2006-01-02", *until); err != nil {
		log.Fatal(err)
	}
	if Until.Before(From) {
		log.Fatal(oaimi.ErrInvalidDateRange)
	}

	if _, err := os.Stat(*cacheDir); os.IsNotExist(err) {
		if err := os.MkdirAll(*cacheDir, 0755); err != nil {
			log.Fatal(err)
		}
	}

	req := oaimi.BatchedRequest{
		Cache: oaimi.Cache{Directory: *cacheDir},
		Request: oaimi.Request{
			Verbose:  *verbose,
			Verb:     "ListRecords",
			Set:      *set,
			Prefix:   *prefix,
			From:     From,
			Until:    Until,
			Endpoint: endpoint,
			MaxRetry: *retry,
		},
	}

	w := bufio.NewWriter(os.Stdout)
	if *root != "" {
		w.WriteString(fmt.Sprintf("<%s>", *root))
	}
	if err = req.Do(w); err != nil {
		log.Fatal(err)
	}
	if *root != "" {
		w.WriteString(fmt.Sprintf("</%s>", *root))
	}
	if err = w.Flush(); err != nil {
		log.Fatal(err)
	}
}
