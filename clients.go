//  Copyright 2015 by Leipzig University Library, http://ub.uni-leipzig.de
//                    The Finc Authors, http://finc.info
//                    Martin Czygan, <martin.czygan@uni-leipzig.de>
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
//+build linux darwin
package oaimi

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/sethgrid/pester"
)

// HttpRequestDoer lets us use pester, DefaultClient or other HTTP client
// implementations interchangably.
type HttpRequestDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client is a simple client, that can turn a OAI request into a OAI response.
type Client struct {
	// client is a delegate for HTTP requests.
	doer HttpRequestDoer
}

// NewClient creates a new OAI client with a user supplied http client, e.g.
// pester.Client, http.DefaultClient.
func NewClientDoer(doer HttpRequestDoer) Client {
	return Client{doer: doer}
}

// NewClient create a default client with resilient HTTP client.
func NewClient() Client {
	c := pester.New()
	c.Timeout = 5 * time.Minute
	c.MaxRetries = 8
	c.Backoff = pester.ExponentialBackoff
	return Client{doer: c}
}

// Do takes an OAI request and turns it into at most one single OAI response.
func (c Client) Do(req Request) (Response, error) {
	var response Response

	link, err := req.URL()
	if err != nil {
		return response, err
	}

	if Verbose {
		log.Println(link)
	}

	hreq, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return response, err
	}
	hreq.Header.Set("User-Agent", UserAgent)
	resp, err := c.doer.Do(hreq)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return response, err
	}
	if response.Error.Code != "" {
		e := response.Error
		return response, OAIError{Code: e.Code, Message: e.Message}
	}

	return response, nil
}

// BatchingClient takes a single OAI request but will do more the one HTTP
// request to fulfill it, if necessary.
type BatchingClient struct {
	// MaxRequests, zero means no limit. Default of 1024 will prevent endless
	// loop due to broken resumptionToken implementations (e.g.
	// http://goo.gl/KFb9iM).
	MaxRequests int
	// client is a our OAI delegate
	client Client
}

// NewBatchingClient returns a client that batches HTTP requests and uses a
// resilient HTTP client.
func NewBatchingClient() BatchingClient {
	return BatchingClient{client: NewClient(), MaxRequests: 1024}
}

// getResumptionToken returns the value of the first found resumptionToken.
func getResumptionToken(resp Response) string {
	// In cases where the request that generated this response did not result
	// in an error or exception condition, the attributes and attribute values
	// of the request element must match the key=value pairs of the protocol
	// request (3.2 XML Response Format).
	switch resp.Request.Verb {
	case "ListIdentifiers":
		return resp.ListIdentifiers.Token.Value
	case "ListRecords":
		return resp.ListRecords.Token.Value
	case "ListSets":
		return resp.ListSets.Token.Value
	}
	return ""
}

// Do will turn a single request into a single response by combining many
// responses into a single one. This is potentially very memory consuming.
func (c *BatchingClient) Do(req Request) (resp Response, err error) {
	resp, err = c.client.Do(req)
	if err != nil {
		return resp, err
	}
	var aggregate = resp
	i := 1
	switch req.Verb {
	case "ListIdentifiers", "ListRecords", "ListSets":
		for {
			if i == c.MaxRequests {
				return aggregate, ErrTooManyRequests
			}
			token := getResumptionToken(resp)
			if token == "" {
				return aggregate, err
			}
			req.ResumptionToken = token
			resp, err = c.client.Do(req)
			if err != nil {
				return aggregate, err
			}
			switch req.Verb {
			case "ListIdentifiers":
				aggregate.ListIdentifiers.Header = append(aggregate.ListIdentifiers.Header,
					resp.ListIdentifiers.Header...)
			case "ListRecords":
				aggregate.ListRecords.Records = append(aggregate.ListRecords.Records,
					resp.ListRecords.Records...)
			case "ListSets":
				aggregate.ListSets.Sets = append(aggregate.ListSets.Sets,
					resp.ListSets.Sets...)
			}
			i++
		}
	}
	return resp, err
}

// WriterClient can execute requests, but writes results to a given writer.
type WriterClient struct {
	// RootTag is used as synthetic root element.
	RootTag string
	// MaxRequests, zero means no limit. Default of 4096 will prevent endless
	// loop due to broken resumptionToken implementations (e.g.
	// http://goo.gl/KFb9iM). Zero means no limit.
	MaxRequests int
	// client is a actual client used for executing the requests.
	client Client
	// w is where the XML gets written.
	w io.Writer
}

func NewWriterClient(w io.Writer) WriterClient {
	return WriterClient{client: NewClient(), w: w, MaxRequests: 16384}
}

func (c WriterClient) writeResponse(resp Response) error {
	b, err := xml.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = c.w.Write(b)
	return err
}

// startDocument will write the root start tag, if one is defined.
func (c WriterClient) startDocument() error {
	if c.RootTag != "" {
		if _, err := c.w.Write([]byte("<" + c.RootTag + ">")); err != nil {
			return err
		}
	}
	return nil
}

// endDocument will close the root tag.
func (c WriterClient) endDocument() error {
	if c.RootTag != "" {
		if _, err := c.w.Write([]byte("</" + c.RootTag + ">")); err != nil {
			return err
		}
	}
	return nil
}

// Do will execute a request and write all XML to the writer.
func (c WriterClient) Do(req Request) error {
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if err := c.startDocument(); err != nil {
		return err
	}
	defer c.endDocument()

	if err := c.writeResponse(resp); err != nil {
		return err
	}
	i := 1
	switch req.Verb {
	case "ListIdentifiers", "ListRecords", "ListSets":
		for {
			if i == c.MaxRequests {
				return ErrTooManyRequests
			}
			token := getResumptionToken(resp)
			if token == "" {
				return nil
			}
			req.ResumptionToken = token
			resp, err = c.client.Do(req)
			if err != nil {
				return err
			}
			if err := c.writeResponse(resp); err != nil {
				return err
			}
			i++
		}
	}
	return nil
}

// CachingClient will write XML to a given writer. This client encapsulates
// cache logic which helps to make subsequent requests fast. A root element is
// optional.
type CachingClient struct {
	// RootTag is an optional root element.
	RootTag string
	// NameSpaces allow to add custom XML namespace declarations to the root element.
	NameSpaces map[string]string
	// CacheDir stores the directory, where all the downloads go.
	CacheDir string
	// w is the target writer, where all content is written.
	w io.Writer
}

// NewCachingClient creates a new client, with a default location for cached
// files. All XML responses will be written to the given io.Writer.
func NewCachingClient(w io.Writer) CachingClient {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return NewCachingClientDir(w, filepath.Join(home, DefaultCacheDir))
}

// NewCachingClient creates a new client, with a default location for cached
// files. All XML responses will be written to the given io.Writer.
func NewCachingClientDir(w io.Writer, dir string) CachingClient {
	defaultns := map[string]string{
		"xsi":    "http://www.w3.org/2001/XMLSchema-instance",
		"dc":     "http://purl.org/dc/elements/1.1/",
		"oai_dc": "http://www.openarchives.org/OAI/2.0/oai_dc/",
	}
	return CachingClient{CacheDir: dir, w: w, NameSpaces: defaultns}
}

// RequestCacheDir returns the cache directory for a given request.
func (c CachingClient) RequestCacheDir(req Request) (string, error) {
	pth, err := c.getCachePath(req)
	if err != nil {
		return "", err
	}
	return path.Dir(pth), nil
}

// getCachePath assembles a destination path for the cache file for a given
// request. This method does not create any file or directory.
func (c CachingClient) getCachePath(req Request) (string, error) {
	ref, err := url.Parse(req.Endpoint)
	if err != nil {
		return "", err
	}
	if ref.Host == "" {
		return "", ErrNoHost
	}
	switch req.Verb {
	case "ListRecords", "ListSets", "ListIdentifiers":
		switch {
		case req.From.IsZero() || req.Until.IsZero():
			return "", ErrMissingFromOrUntil
		default:
			name := fmt.Sprintf("%s-%s.xml.gz", req.From.Format("2006-01-02"), req.Until.Format("2006-01-02"))
			sub := path.Join(ref.Host, ref.Path, req.Verb, req.Prefix)
			return filepath.Join(c.CacheDir, sub, name), nil
		}
	}
	return "", ErrCannotCreatePath
}

// startDocument inserts a root tag, if given.
func (c CachingClient) startDocument() error {
	if c.RootTag == "" {
		return nil
	}
	var nslist []string
	for k, v := range c.NameSpaces {
		nslist = append(nslist, fmt.Sprintf(`xmlns:%s="%s"`, k, v))
	}
	namespaces := strings.Join(nslist, " ")
	tag := fmt.Sprintf("<%s %s>", c.RootTag, namespaces)
	if _, err := c.w.Write([]byte(tag)); err != nil {
		return err
	}
	return nil
}

// endDocument closes the root tag.
func (c CachingClient) endDocument() error {
	if c.RootTag == "" {
		return nil
	}
	if _, err := c.w.Write([]byte("</" + c.RootTag + ">")); err != nil {
		return err
	}
	return nil
}

// maybeRetrieve retrieves and stores the response for a given request, if it
// is not already cached. Returns the cache filename and any error.
func (c CachingClient) maybeRetrieve(req Request) (fn string, err error) {
	fn, err = c.getCachePath(req)
	if err != nil {
		return fn, err
	}
	// retrieve records if we don't already have
	if _, err := os.Stat(fn); os.IsNotExist(err) {
		file := CreateMaybeCompressedFile(fn)
		client := NewWriterClient(file)
		if err := client.Do(req); err != nil {
			switch err := err.(type) {
			case OAIError:
				if err.Code != "noRecordsMatch" {
					return fn, err
				}
			default:
				return fn, err
			}
		}
		if err := file.Close(); err != nil {
			return fn, err
		}
	}
	return fn, nil
}

// Do executes a given request. If the request is not yet cached, the content
// is retrieved and persisted. Requests are internally split up into weekly
// windows to reduce load and to latency in case of errors.
func (c CachingClient) Do(req Request) error {
	c.startDocument()
	defer c.endDocument()

	switch req.Verb {
	case "Identify", "ListMetadataFormats", "ListSets":
		client := NewWriterClient(c.w)
		return client.Do(req)
	case "ListRecords", "ListIdentifiers":
		req.UseDefaults()
		windows := Window{From: req.From, Until: req.Until}.Weekly()
		for _, w := range windows {
			r := Request{
				Endpoint: req.Endpoint,
				Verb:     req.Verb,
				Prefix:   req.Prefix,
				Set:      req.Set,
				From:     w.From,
				Until:    w.Until,
			}

			filename, err := c.maybeRetrieve(r)
			if err != nil {
				return err
			}
			file, err := OpenMaybeCompressedFile(filename)
			if err != nil {
				return err
			}
			if _, err = io.Copy(c.w, file); err != nil {
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}
