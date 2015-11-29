// Package next should simplify building OAI apps.
package next

import (
	"bufio"
	"compress/gzip"
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
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/sethgrid/pester"
)

// Version
const Version = "0.2.0"

var (
	ErrNoEndpoint         = errors.New("request: an endpoint is required")
	ErrNoVerb             = errors.New("no verb")
	ErrBadVerb            = errors.New("bad verb")
	ErrCannotCreatePath   = errors.New("cannot create path")
	ErrNoHost             = errors.New("no host")
	ErrMissingFromOrUntil = errors.New("missing from or until")

	// UserAgent to use for requests
	UserAgent = fmt.Sprintf("oaimi/%s (https://github.com/miku/oaimi)", Version)

	// DefaultEarliestDate is used, if the repository does not supply one.
	DefaultEarliestDate = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	// DefaultFormat should be supported by most endpoints.
	DefaultFormat = "oai_dc"
)

var (
	// DefaultClient should suffice for most use cases.
	DefaultClient = NewClient()
	// OAIVerbs (4. Protocol Requests and Responses)
	OAIVerbs = map[string]bool{
		"Identify":            true,
		"ListIdentifiers":     true,
		"ListSets":            true,
		"ListMetadataFormats": true,
		"ListRecords":         true,
		"GetRecord":           true,
	}
)

// OAIError wraps OAI error codes and messages.
type OAIError struct {
	Code    string
	Message string
}

// Error to satisfy interface.
func (e OAIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Request can hold any parameter, that you want to send to an OAI server.
type Request struct {
	Endpoint        string
	Verb            string
	From            time.Time
	Until           time.Time
	Set             string
	Prefix          string
	Identifier      string
	ResumptionToken string
}

// useDefaults will fill in default values for From, Until and Prefix if
// they are missing.
func useDefaults(r Request) Request {
	if r.From.IsZero() {
		c := NewClient()
		req := Request{Verb: "Identify", Endpoint: r.Endpoint}
		resp, err := c.Do(req)
		switch {
		case err != nil, resp.Identify.EarliestDatestamp == "", len(resp.Identify.EarliestDatestamp) < 10:
			r.From = DefaultEarliestDate
		default:
			r.From, err = time.Parse("2006-01-02", resp.Identify.EarliestDatestamp[:10])
			if err != nil {
				r.From = DefaultEarliestDate
			}
		}
	}
	if r.Until.IsZero() {
		r.Until = time.Now()
	}
	if r.Prefix == "" {
		r.Prefix = DefaultFormat
	}
	return r
}

// URL returns the absolute URL for a given request. Catches basic errors like
// missing endpoint or bad verb.
func (r Request) URL() (s string, err error) {
	if r.Endpoint == "" {
		return s, ErrNoEndpoint
	}
	if r.Verb == "" {
		return s, ErrNoVerb
	}
	if _, found := OAIVerbs[r.Verb]; !found {
		return s, ErrBadVerb
	}

	values := url.Values{}
	values.Add("verb", r.Verb)

	// Collectively these requests are called list requests (3.5):
	// ListIdentifiers, ListRecords, ListSets
	if r.ResumptionToken != "" {
		// An exclusive argument with a value that is the flow control token.
		values.Add("resumptionToken", r.ResumptionToken)
		return fmt.Sprintf("%s?%s", r.Endpoint, values.Encode()), nil
	}

	maybeAdd := func(k, v string) {
		if v != "" {
			values.Add(k, v)
		}
	}
	switch r.Verb {
	case "ListRecords", "ListIdentifiers":
		maybeAdd("from", r.From.Format("2006-01-02"))
		maybeAdd("until", r.Until.Format("2006-01-02"))
		switch r.Verb {
		case "ListRecords":
			maybeAdd("set", r.Set)
			maybeAdd("metadataPrefix", r.Prefix)
		}
	case "GetRecord":
		maybeAdd("identifier", r.Identifier)
	}
	return fmt.Sprintf("%s?%s", r.Endpoint, values.Encode()), nil
}

// makeCachePath turns a request into a uniq string, that is safe to use a
// path component.
func makeCachePath(req Request) (string, error) {
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
			return path.Join(ref.Host, ref.Path, req.Verb, req.Prefix,
				fmt.Sprintf("%s-%s.xml", req.From.Format("2006-01-02"), req.Until.Format("2006-01-02"))), nil
		}
	case "Identify":
		return path.Join(ref.Host, ref.Path, req.Verb, "Identify"), nil
	}
	return "", ErrCannotCreatePath
}

// resumptionToken is part of OAI flow control (3.5)
type resumptionToken struct {
	//
	Value string `xml:",chardata"`
	// The following optional attributes may be included as part of the
	// resumptionToken element along with the resumptionToken itself. A
	// UTCdatetime indicating when the resumptionToken ceases to be valid.
	ExpirationDate string `xml:"expirationDate"`
	// A count of the number of elements of the complete list thus far
	// returned (i.e. cursor starts at 0).
	Cursor string `xml:"cursor,attr"`
	// An integer indicating the cardinality of the complete list. The value
	// of completeListSize may be only an estimate of the actual cardinality
	// of the complete list and may be revised during the list request
	// sequence.
	CompleteListSize string `xml:"completeListSize,attr"`
}

// header is the main response of ListIdentifiers requests and also
// transmitted in ListRecords.
type header struct {
	Identifier string `xml:"identifier"`
	Datestamp  string `xml:"datestamp"`
	Set        string `xml:"setSpec"`
}

// Response can hold most answers to an request to a OAI server.
type Response struct {
	xml.Name `xml:"response"`
	Date     string `xml:"responseDate"`
	Request  struct {
		Verb     string `xml:"verb,attr"`
		Endpoint string `xml:",chardata"`
	} `xml:"request,omitempty"`
	ListIdentifiers struct {
		Header []header        `xml:"header"`
		Token  resumptionToken `xml:"resumptionToken"`
	} `xml:"ListIdentifiers,omitempty"`
	ListMetadataFormats struct {
		xml.Name `xml:"ListMetadataFormats" json:"formats"`
		Formats  []struct {
			Prefix string `xml:"metadataPrefix" json:"prefix"`
			Schema string `xml:"schema" json:"schema"`
		} `xml:"metadataFormat" json:"format"`
	} `xml:"ListMetadataFormats,omitempty" json:"sets"`
	ListSets struct {
		Sets []struct {
			Spec        string `xml:"setSpec" json:"spec,omitempty"`
			Name        string `xml:"setName" json:"name,omitempty"`
			Description string `xml:"setDescription>dc>description" json:"description,omitempty"`
		} `xml:"set" json:"set"`
		Token resumptionToken `xml:"resumptionToken"`
	} `xml:"ListSets,omitempty" json:"sets"`
	ListRecords struct {
		Records []struct {
			Header   header `xml:"header"`
			Metadata struct {
				Verbatim string `xml:",innerxml"`
			} `xml:"metadata"`
		} `xml:"record"`
		Token resumptionToken `xml:"resumptionToken"`
	} `xml:"ListRecords,omitempty"`
	Identify struct {
		Name              string `xml:"repositoryName,omitempty" json:"name"`
		URL               string `xml:"baseURL,omitempty" json:"url"`
		Version           string `xml:"protocolVersion,omitempty" json:"version"`
		AdminEmail        string `xml:"adminEmail,omitempty" json:"email"`
		EarliestDatestamp string `xml:"earliestDatestamp,omitempty" json:"earliest"`
		DeletePolicy      string `xml:"deletedRecord,omitempty" json:"delete"`
		Granularity       string `xml:"granularity,omitempty" json:"granularity"`
		Description       struct {
			Identifier struct {
				Scheme               string `xml:"scheme,omitempty" json:"scheme,omitempty"`
				RepositoryIdentifier string `xml:"repositoryIdentifier,omitempty" json:"repositoryIdentifier,omitempty"`
				Delimiter            string `xml:"delimiter,omitempty" json:"delimiter,omitempty"`
				SampleIdentifier     string `xml:"sampleIdentifier,omitempty" json:"sampleIdentifier,omitempty"`
			} `xml:"oai-identifier,omitempty" json:"identifier,omitempty"`
		} `xml:"description,omitempty" json:"description,omitempty"`
	} `xml:"Identify,omitempty"`
	Error struct {
		Code    string `xml:"code,attr"`
		Message string `xml:",chardata"`
	} `xml:"error"`
}

// HttpRequestDoer let's us use pester, DefaultClient or others interchangably.
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
	c.Timeout = 60 * time.Second
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

	log.Println(link)

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
	// client is a our OAI delegate
	client Client
}

// NewBatchingClient returns a client that batches HTTP requests and uses a
// resilient HTTP client.
func NewBatchingClient() BatchingClient {
	return BatchingClient{client: NewClient()}
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
	switch req.Verb {
	case "ListIdentifiers", "ListRecords", "ListSets":
		for {
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
		}
	}
	return resp, err
}

// WriterClient can execute requests, but writes results to a given writer.
type WriterClient struct {
	RootTag string
	client  Client
	w       io.Writer
}

func NewWriterClient(w io.Writer) WriterClient {
	return WriterClient{client: NewClient(), w: w}
}

func (c WriterClient) writeResponse(resp Response) error {
	b, err := xml.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = c.w.Write(b)
	return err
}

func (c WriterClient) startDocument() error {
	if c.RootTag != "" {
		if _, err := c.w.Write([]byte("<" + c.RootTag + ">")); err != nil {
			return err
		}
	}
	return nil
}

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
	switch req.Verb {
	case "ListIdentifiers", "ListRecords", "ListSets":
		for {
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
		}
	}
	return nil
}

// CachingClient will write XML to a given writer. This client encapsulates
// cache logic which helps to make subsequent requests fast. A root element is
// optional.
type CachingClient struct {
	RootTag  string
	CacheDir string
	w        io.Writer
}

// NewCachingClient creates a new client, with a default location for cached
// files. All XML responses will be written to the given io.Writer.
func NewCachingClient(w io.Writer) CachingClient {
	home, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return CachingClient{RootTag: "collection", CacheDir: filepath.Join(home, ".oaimicache"), w: w}
}

// makeCachePath assembles a destination path for the cache file for a given
// request. This method does not create any file or directory.
func (c CachingClient) makeCachePath(req Request) (string, error) {
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

// ensureDir ensures a path exists and is a directory.
func ensureDir(dir string) error {
	fi, err := os.Stat(dir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else {
		if !fi.IsDir() {
			return fmt.Errorf("%s is not a directory", dir)
		}
	}
	return nil
}

// startDocument inserts a root tag, if given.
func (c CachingClient) startDocument() error {
	if c.RootTag != "" {
		if _, err := c.w.Write([]byte("<" + c.RootTag + ">")); err != nil {
			return err
		}
	}
	return nil
}

// endDocument closes the root tag.
func (c CachingClient) endDocument() error {
	if c.RootTag != "" {
		if _, err := c.w.Write([]byte("</" + c.RootTag + ">")); err != nil {
			return err
		}
	}
	return nil
}

// do executes a oai request and will create a compressed file under the
// client's CacheDir.
func (c CachingClient) do(req Request) error {
	file, err := ioutil.TempFile("", "oaimi-")
	if err != nil {
		return err
	}
	// move temporary file into place
	defer func() error {
		dst, err := c.makeCachePath(req)
		if err != nil {
			return err
		}
		dir := path.Dir(dst)
		if err := ensureDir(dir); err != nil {
			return err
		}
		return os.Rename(file.Name(), dst)

	}()
	defer file.Close()

	bw := bufio.NewWriter(file)
	defer bw.Flush()

	gz := gzip.NewWriter(bw)
	defer gz.Close()

	client := NewWriterClient(gz)
	if err := client.Do(req); err != nil {
		switch err := err.(type) {
		case OAIError:
			if err.Code == "noRecordsMatch" {
				log.Println("no records")
			}
		default:
			return err
		}
	}
	return nil
}

// copyFile moves the file content to the writer.
func (c CachingClient) copyFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() error { return file.Close() }()

	gz, err := gzip.NewReader(bufio.NewReader(file))
	if err != nil {
		return err
	}
	defer func() error { return gz.Close() }()

	_, err = io.Copy(c.w, gz)
	return err
}

// Do executes a given request. If the request is not yet cached, the content
// is retrieved and persisted. Requests are internally split up into weekly
// windows to reduce load and to latency in case of errors.
func (c CachingClient) Do(req Request) error {
	if err := c.startDocument(); err != nil {
		return err
	}
	defer func() error {
		return c.endDocument()
	}()

	switch req.Verb {
	case "Identify", "ListMetadataFormats", "ListSets":
		wc := WriterClient{client: NewClient(), w: c.w}
		return wc.Do(req)
	case "ListRecords", "ListIdentifiers":
		req := useDefaults(req)
		windows, err := Window{From: req.From, Until: req.Until}.Weekly()
		if err != nil {
			return err
		}
		for _, w := range windows {
			r := Request{
				Endpoint: req.Endpoint,
				Verb:     req.Verb,
				Prefix:   req.Prefix,
				Set:      req.Set,
				From:     w.From,
				Until:    w.Until,
			}
			filename, err := c.makeCachePath(r)
			if err != nil {
				return err
			}
			if _, err := os.Stat(filename); os.IsNotExist(err) {
				if err := c.do(r); err != nil {
					return err
				}
			}
			if err := c.copyFile(filename); err != nil {
				return err
			}
		}
	}
	return nil
}
