// Package next should simplify building OAI apps.
package next

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/sethgrid/pester"
)

// Version
const Version = "0.2.0"

var (
	ErrNoEndpoint = errors.New("request: an endpoint is required")
	ErrNoVerb     = errors.New("no verb")
	ErrBadVerb    = errors.New("bad verb")

	// UserAgent to use for requests
	UserAgent = fmt.Sprintf("oaimi/%s (https://github.com/miku/oaimi)", Version)
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
	From            string
	Until           string
	Set             string
	Prefix          string
	Identifier      string
	ResumptionToken string
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

	maybeAdd("from", r.From)
	maybeAdd("until", r.Until)
	maybeAdd("set", r.Set)
	maybeAdd("metadataPrefix", r.Prefix)
	maybeAdd("identifier", r.Identifier)
	return fmt.Sprintf("%s?%s", r.Endpoint, values.Encode()), nil
}

// makeCachePath turns a request into a uniq string, that is safe to use a
// path component.
func makeCachePath(req Request) string {
	return ""
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
	identifier string `xml:"identifier"`
	datestamp  string `xml:"datestamp"`
	set        string `xml:"setSpec"`
}

// Response can hold any answer to an request to a OAI server.
type Response struct {
	Date    string `xml:"responseDate"`
	Request struct {
		Verb string `xml:"verb,attr"`
	} `xml:"request"`
	ListIdentifiers struct {
		Header []header        `xml:"header"`
		Token  resumptionToken `xml:"resumptionToken"`
	} `xml:"ListIdentifiers"`
	ListMetadataFormats struct {
		xml.Name `xml:"ListMetadataFormats" json:"formats"`
		Formats  []struct {
			Prefix string `xml:"metadataPrefix" json:"prefix"`
			Schema string `xml:"schema" json:"schema"`
		} `xml:"metadataFormat" json:"format"`
	}
	ListSets struct {
		Sets []struct {
			Spec        string `xml:"setSpec" json:"spec,omitempty"`
			Name        string `xml:"setName" json:"name,omitempty"`
			Description string `xml:"setDescription>dc>description" json:"description,omitempty"`
		} `xml:"set" json:"set"`
		Token resumptionToken `xml:"resumptionToken"`
	} `xml:"ListSets" json:"sets"`
	ListRecords struct {
		Records []struct {
			Header   header `xml:"header"`
			Metadata string `xml:",innerxml"`
		} `xml:"record"`
		Token resumptionToken `xml:"resumptionToken"`
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
func NewClientDoer(doer HttpRequestDoer) *Client {
	return &Client{doer: doer}
}

// NewClient create a default client with resilient HTTP client.
func NewClient() *Client {
	c := pester.New()
	c.Timeout = 60 * time.Second
	c.MaxRetries = 8
	c.Backoff = pester.ExponentialBackoff
	return &Client{doer: c}
}

// Do takes an OAI request and turns it into at most one single OAI response.
func (c *Client) Do(req Request) (Response, error) {
	var response Response

	link, err := req.URL()
	if err != nil {
		return response, err
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
