package oaimi

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"path"
	"time"
)

var (
	ErrNoEndpoint         = errors.New("request: an endpoint is required")
	ErrNoVerb             = errors.New("no verb")
	ErrBadVerb            = errors.New("bad verb")
	ErrCannotCreatePath   = errors.New("cannot create path")
	ErrNoHost             = errors.New("no host")
	ErrMissingFromOrUntil = errors.New("missing from or until")
	ErrTooManyRequests    = errors.New("too many requests")

	// UserAgent to use for requests
	UserAgent = fmt.Sprintf("oaimi/%s (https://github.com/miku/oaimi)", Version)

	// DefaultEarliestDate is used, if the repository does not supply one.
	DefaultEarliestDate = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	// DefaultFormat should be supported by most endpoints.
	DefaultFormat = "oai_dc"
	// DefaultCacheDir
	DefaultCacheDir = ".oaimicache"
	// Verbose logs actions
	Verbose = false
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
func (r *Request) UseDefaults() {
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
}

// URL returns the absolute URL for a given request. Catches basic errors like
// missing endpoint or bad verb.
func (r *Request) URL() (s string, err error) {
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

	maybeAdd := func(k string, v interface{}) {
		switch val := v.(type) {
		case time.Time:
			if !val.IsZero() {
				values.Add(k, val.Format("2006-01-02"))
			}
		case string:
			if val != "" {
				values.Add(k, val)
			}
		default:
			panic(fmt.Sprintf("maybeAdd cannot handle %T", v))
		}
	}
	switch r.Verb {
	case "ListRecords", "ListIdentifiers":
		maybeAdd("from", r.From)
		maybeAdd("until", r.Until)
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
		Name              string `xml:"repositoryName,omitempty" json:"name,omitempty"`
		URL               string `xml:"baseURL,omitempty" json:"url,omitempty"`
		Version           string `xml:"protocolVersion,omitempty" json:"version,omitempty"`
		AdminEmail        string `xml:"adminEmail,omitempty" json:"email,omitempty"`
		EarliestDatestamp string `xml:"earliestDatestamp,omitempty" json:"earliest,omitempty"`
		DeletePolicy      string `xml:"deletedRecord,omitempty" json:"delete,omitempty"`
		Granularity       string `xml:"granularity,omitempty" json:"granularity,omitempty"`
		Description       struct {
			Friends    []string `xml:"friends>baseURL,omitempty" json:"friends,omitempty"`
			Identifier struct {
				Scheme               string `xml:"scheme,omitempty" json:"scheme,omitempty"`
				RepositoryIdentifier string `xml:"repositoryIdentifier,omitempty" json:"repositoryIdentifier,omitempty"`
				Delimiter            string `xml:"delimiter,omitempty" json:"delimiter,omitempty"`
				SampleIdentifier     string `xml:"sampleIdentifier,omitempty" json:"sampleIdentifier,omitempty"`
			} `xml:"oai-identifier,omitempty" json:"identifier,omitempty"`
		} `xml:"description,omitempty" json:"description,omitempty"`
	} `xml:"Identify,omitempty" json:"identity,omitempty"`
	Error struct {
		Code    string `xml:"code,attr"`
		Message string `xml:",chardata"`
	} `xml:"error"`
}
