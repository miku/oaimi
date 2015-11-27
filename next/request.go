// Package next should simplify building OAI apps.
package next

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/sethgrid/pester"
)

// Version
const Version = "0.1.10"

var (
	ErrNoEndpoint = errors.New("request: an endpoint is required")
	ErrNoVerb     = errors.New("no verb")
	ErrBadVerb    = errors.New("bad verb")

	// UserAgent to use for requests
	UserAgent = fmt.Sprintf("oaimi/%s (https://github.com/miku/oaimi)", Version)
)

var (
	// DefaultClient should suffice for most use cases.
	DefaultClient = Client{MaxRetry: 10, Timeout: 60 * time.Second}
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

// Response can hold any answer to an request to a OAI server.
type Response struct {
	Date    string `xml:"responseDate"`
	Request struct {
		Verb string `xml:"verb,attr"`
	} `xml:"request"`
	ListIdentifiers struct {
		Raw   string          `xml:",innerxml"`
		Token resumptionToken `xml:"resumptionToken"`
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
		Raw   string          `xml:",innerxml"`
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

// Client is a simple client, that can turn a OAI request into a OAI response.
// Supports retries with exponential backoff.
type Client struct {
	Verbose  bool
	MaxRetry int
	Timeout  time.Duration
}

func (c Client) Do(req Request) (Response, error) {
	if c.Verbose {
		log.Println(req.URL())
	}

	client := pester.New()
	client.Timeout = c.Timeout
	client.MaxRetries = c.MaxRetry
	client.Backoff = pester.ExponentialBackoff

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
	resp, err := client.Do(hreq)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return response, err
	}
	if response.Error.Code != "" {
		return response, OAIError{Code: response.Error.Code, Message: response.Error.Message}
	}

	return response, nil
}
