//  Copyright 2015 by Leipzig University Library, http://ub.uni-leipzig.de
//                 by The Finc Authors, http://finc.info
//                 by Martin Czygan, <martin.czygan@uni-leipzig.de>
//
// This file is part of some open source application.
//
// Some open source application is free software: you can redistribute
// it and/or modify it under the terms of the GNU General Public
// License as published by the Free Software Foundation, either
// version 3 of the License, or (at your option) any later version.
//
// Some open source application is distributed in the hope that it will
// be useful, but WITHOUT ANY WARRANTY; without even the implied warranty
// of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Foobar.  If not, see <http://www.gnu.org/licenses/>.
//
// @license GPL-3.0+ <http://spdx.org/licenses/GPL-3.0+>
//
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
	timeout := flag.Duration("timeout", 60*time.Second, "request timeout")

	flag.Parse()

	if *showVersion {
		fmt.Println(oaimi.Version)
		os.Exit(0)
	}

	if flag.NArg() < 1 {
		log.Fatal("URL to OAI endpoint required")
	}

	if *retry < 1 {
		log.Fatal("retry > 0 required")
	}

	endpoint := flag.Arg(0)

	if _, err := url.Parse(endpoint); err != nil {
		log.Fatal("endpoint is not an URL")
	}

	if *identify {
		var req oaimi.Request
		var err error

		req = oaimi.Request{Endpoint: endpoint, Verb: "Identify", Verbose: *verbose, MaxRetry: *retry, Timeout: *timeout}
		responseIdentify, err := req.DoOne()
		if err != nil {
			log.Fatal(err)
		}

		if responseIdentify.Identify.URL == "" {
			log.Fatal("no URL in identify, possible broken repository")
		}

		req = oaimi.Request{Endpoint: endpoint, Verb: "ListMetadataFormats", Verbose: *verbose, MaxRetry: *retry, Timeout: *timeout}
		responseFormats, err := req.DoOne()
		if err != nil {
			log.Fatal(err)
		}

		req = oaimi.Request{Endpoint: endpoint, Verb: "ListSets", Verbose: *verbose, MaxRetry: *retry, Timeout: *timeout}
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
				Timeout:  *timeout,
			},
		}
		fmt.Println(filepath.Dir(req.Path()))
		os.Exit(0)
	}

	var From, Until time.Time

	if *from == "" || *fromEarliest {
		req := oaimi.Request{Endpoint: endpoint, Verb: "Identify", Verbose: *verbose, MaxRetry: *retry, Timeout: *timeout}
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
			Timeout:  *timeout,
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
