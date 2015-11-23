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
package oaimi

import (
	"bufio"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/jinzhu/now"
	"github.com/sethgrid/pester"
)

// Version
const Version = "0.1.10"

var (
	ErrInvalidDateRange = errors.New("invalid date range")
	ErrVerbNotSupported = errors.New("verb not supported by client")

	UserAgent = fmt.Sprintf("oaimi/%s (https://github.com/miku/oaimi)", Version)
)

// OAIError wraps OAI error codes and messages.
type OAIError struct {
	Code    string
	Message string
}

// Error to satisfy interface.
func (e OAIError) Error() string {
	return fmt.Sprintf("%s %s", e.Code, e.Message)
}

// Values is a thin wrapper around url.Values.
type Values struct {
	url.Values
}

// NewValues returns a new empty struct.
func NewValues() Values {
	return Values{url.Values{}}
}

// AddIfExists add a key value pair only if value is nonempty.
func (v Values) AddIfExists(key, value string) {
	if value != "" {
		v.Add(key, value)
	}
}

// Cache is a simple cache configuration.
type Cache struct {
	Directory string
}

// Response is a minimal response object, which currently knows only about
// ListRecords, Identify and errors.
type Response struct {
	Date    string `xml:"responseDate"`
	Request struct {
		Verb string `xml:"verb,attr"`
	} `xml:"request"`
	ListMetadataFormats struct {
		Formats []struct {
			Prefix string `xml:"metadataPrefix" json:"prefix"`
			Schema string `xml:"schema" json:"schema"`
		} `xml:"metadataFormat" json:"format"`
	} `xml:"ListMetadataFormats" json:"formats"`
	ListSets struct {
		Sets []struct {
			Spec        string `xml:"setSpec" json:"spec,omitempty"`
			Name        string `xml:"setName" json:"name,omitempty"`
			Description string `xml:"setDescription>dc>description" json:"description,omitempty"`
		} `xml:"set" json:"set"`
	} `xml:"ListSets" json:"sets"`
	ListRecords struct {
		Raw   string `xml:",innerxml"`
		Token struct {
			Value  string `xml:",chardata"`
			Cursor string `xml:"cursor,attr"`
			Size   string `xml:"completeListSize,attr"`
		} `xml:"resumptionToken"`
	} `xml:"ListRecords"`
	Identify struct {
		Name              string `xml:"repositoryName" json:"name"`
		URL               string `xml:"baseURL" json:"url"`
		Version           string `xml:"protocolVersion" json:"version"`
		AdminEmail        string `xml:"adminEmail" json:"email"`
		EarliestDatestamp string `xml:"earliestDatestamp" json:"earliest"`
		DeletePolicy      string `xml:"deletedRecord" json:"delete"`
		Granularity       string `xml:"granularity" json:"granularity"`
	} `xml:"Identify"`
	Error struct {
		Code    string `xml:"code,attr"`
		Message string `xml:",chardata"`
	} `xml:"error"`
}

// Request represents an OAI request, which might take multiple HTTP requests
// to fulfill.
type Request struct {
	Endpoint        string
	Verb            string
	From            time.Time
	Until           time.Time
	Set             string
	Prefix          string
	ResumptionToken string
	Verbose         bool
	MaxRetry        int
	Timeout         time.Duration
}

// CachedRequest can serve content from HTTP or a local Cache.
type CachedRequest struct {
	Cache
	Request
}

// BatchedRequest will split up the request internally into monthly batches.
// This provides the real caching value, since this implements incremental
// harvesting.
type BatchedRequest struct {
	Cache
	Request
}

// URL returns the full URL for this request. A resumptionToken will suppress
// some other parameters.
func (r Request) URL() string {
	vals := NewValues()
	vals.AddIfExists("verb", r.Verb)
	if r.ResumptionToken == "" {
		if r.Verb == "ListRecords" {
			vals.AddIfExists("from", r.From.Format("2006-01-02"))
			vals.AddIfExists("until", r.Until.Format("2006-01-02"))
			vals.AddIfExists("metadataPrefix", r.Prefix)
			vals.AddIfExists("set", r.Set)
		}
	} else {
		vals.Add("resumptionToken", r.ResumptionToken)
	}
	return fmt.Sprintf("%s?%s", r.Endpoint, vals.Encode())
}

// DoOne executes a single request only.
func (req Request) DoOne() (Response, error) {
	var response Response
	if req.Verbose {
		log.Println(req.URL())
	}

	client := pester.New()
	client.Timeout = req.Timeout
	client.MaxRetries = req.MaxRetry
	client.Backoff = pester.ExponentialBackoff

	httpreq, err := http.NewRequest("GET", req.URL(), nil)
	if err != nil {
		return response, err
	}
	httpreq.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(httpreq)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	decoder := xml.NewDecoder(resp.Body)
	decoder.Decode(&response)

	if response.Error.Code != "" {
		return response, OAIError{Code: response.Error.Code, Message: response.Error.Message}
	}

	return response, nil
}

// Do will execute one or more HTTP requests to fullfil this OAI request. The
// record metadata XML is written verbatim to the given io.Writer.
func (req Request) Do(w io.Writer) error {
	for {
		if req.Verbose {
			log.Println(req.URL())
		}

		client := pester.New()
		client.Timeout = req.Timeout
		client.MaxRetries = req.MaxRetry
		client.Backoff = pester.ExponentialBackoff

		httpreq, err := http.NewRequest("GET", req.URL(), nil)
		if err != nil {
			return err
		}
		httpreq.Header.Set("User-Agent", UserAgent)
		resp, err := client.Do(httpreq)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var response Response
		decoder := xml.NewDecoder(resp.Body)
		decoder.Decode(&response)

		if response.Error.Code != "" {
			return OAIError{Code: response.Error.Code, Message: response.Error.Message}
		}

		if _, err = w.Write([]byte(response.ListRecords.Raw)); err != nil {
			return err
		}
		if response.ListRecords.Token.Value == "" {
			break
		}
		req.ResumptionToken = response.ListRecords.Token.Value
	}
	return nil
}

// IsCached returns true, if this request has been executed successfully in the past.
func (r CachedRequest) IsCached() bool {
	if _, err := os.Stat(r.Path()); os.IsNotExist(err) {
		return false
	}
	return true
}

// Fingerprint returns a encoded version of the full endpoint and the set.
func (r CachedRequest) Fingerprint() string {
	return base64.RawStdEncoding.EncodeToString([]byte(fmt.Sprintf("%s#%s", r.Endpoint, r.Set)))
}

// Filename returns the filename for a request. It only carries date
// information.
func (r CachedRequest) Filename() string {
	return fmt.Sprintf("%s-%s.xml.gz", r.From.Format("2006-01-02"), r.Until.Format("2006-01-02"))
}

// Path returns the absolute path to the cache file for a single OAI request.
func (r CachedRequest) Path() string {
	u, _ := url.Parse(r.Endpoint)
	return path.Join(r.Cache.Directory, u.Host, r.Prefix, r.Fingerprint(), r.Filename())
}

// Do abstracts from the actual access, cache or HTTP. All OAI errors are
// returned back, except noRecordsMatch, which is used to indicate a zero
// result set.
func (r CachedRequest) Do(w io.Writer) error {
	if !r.IsCached() {

		file, err := ioutil.TempFile("", "oaimi-")
		if err != nil {
			return err
		}
		if r.Request.Verbose {
			log.Printf("tmp@%s", file.Name())
		}
		gw := gzip.NewWriter(file)
		bw := bufio.NewWriter(gw)

		if err := r.Request.Do(bw); err != nil {
			switch t := err.(type) {
			case OAIError:
				if t.Code != "noRecordsMatch" {
					return err
				}
			default:
				return err
			}
		}

		bw.Flush()
		gw.Close()
		file.Close()

		dir := filepath.Dir(r.Path())
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
		if err := os.Rename(file.Name(), r.Path()); err != nil {
			return err
		}
	}

	file, err := os.Open(r.Path())
	if err != nil {
		return err
	}
	rdr, err := gzip.NewReader(bufio.NewReader(file))
	if err != nil {
		return err
	}
	if _, err := io.Copy(w, rdr); err != nil {
		return err
	}
	return nil
}

// Do runs batched requests over a range. All metadata gets written to the
// given writer.
func (r BatchedRequest) Do(w io.Writer) error {
	intervals, err := MonthlyDateRange(r.From, r.Until)
	if err != nil {
		return err
	}
	for _, interval := range intervals {
		req := CachedRequest{
			Cache: Cache{Directory: r.Cache.Directory},
			Request: Request{
				Verbose:  r.Verbose,
				Verb:     r.Verb,
				Set:      r.Set,
				Prefix:   r.Prefix,
				From:     interval.From,
				Until:    interval.Until,
				Endpoint: r.Endpoint,
				MaxRetry: r.MaxRetry,
			},
		}
		if err := req.Do(w); err != nil {
			return err
		}
	}
	return nil
}

// DateRange represent a span of time.
type DateRange struct {
	From  time.Time
	Until time.Time
}

// RangeSplitter returns a list of DateRange values covering a data range in
// monthly intvervals.
func MonthlyDateRange(from, until time.Time) ([]DateRange, error) {
	var ranges []DateRange
	var start, end time.Time

	if from.After(until) {
		return ranges, ErrInvalidDateRange
	}

	for {
		t := now.New(from)
		if len(ranges) == 0 {
			start = t.BeginningOfDay()
		} else {
			start = t.BeginningOfMonth()
		}
		end = t.EndOfMonth()
		if end.After(until) {
			ranges = append(ranges, DateRange{From: start,
				Until: now.New(until).EndOfDay()})
			break
		}
		ranges = append(ranges, DateRange{From: start, Until: end})
		from = end.Add(24 * time.Hour)
	}
	return ranges, nil
}

// repositoryInfo groups some repository info.
type repositoryInfo struct {
	Identify Response
	Formats  Response
	Sets     Response
}

// MarshalJSON make the output more compact.
func (ri repositoryInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"identify": ri.Identify.Identify,
		"formats":  ri.Formats.ListMetadataFormats.Formats,
		"sets":     ri.Sets.ListSets.Sets,
	})
}

// RepositoryInfo returns a compact repositoryInfo object. Runs
// requests on the repo in parallel.
func RepositoryInfo(r Request) (repositoryInfo, error) {
	worker := func(req Request, ch chan Response, wg *sync.WaitGroup) {
		defer wg.Done()
		resp, err := req.DoOne()
		switch err := err.(type) {
		default:
			log.Fatal(err)
		case nil:
			// pass
		case OAIError:
			if err.Code == "noSetHierarchy" {
				log.Println(err)
			} else {
				log.Fatal(err)
			}
		}
		ch <- resp
	}

	ch := make(chan Response)
	infoc := make(chan repositoryInfo)
	var wg sync.WaitGroup

	go func() {
		ri := repositoryInfo{}
		for resp := range ch {
			switch resp.Request.Verb {
			case "Identify", "":
				ri.Identify = resp
			case "ListMetadataFormats":
				ri.Formats = resp
			case "ListSets":
				ri.Sets = resp
			default:
				log.Fatalf("invalid response: %v", resp)
			}
		}
		infoc <- ri
	}()

	verbs := []string{"Identify", "ListMetadataFormats", "ListSets"}

	for _, v := range verbs {
		wg.Add(1)
		verb := v
		req := Request{
			Endpoint: r.Endpoint,
			Verb:     verb,
			Verbose:  r.Verbose,
			MaxRetry: r.MaxRetry,
			Timeout:  r.Timeout,
		}
		go worker(req, ch, &wg)
	}

	wg.Wait()
	close(ch)

	return <-infoc, nil
}
