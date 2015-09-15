package oaimi

import (
	"bufio"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/jinzhu/now"
	"github.com/sethgrid/pester"
)

const Version = "0.1.1"

var (
	ErrInvalidDateRange = errors.New("invalid date range")
	ErrRetriesExceeded  = errors.New("retried and failed too many times")
)

type OAIError struct {
	Code    string
	Message string
}

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
// TODO: make cache more transparent.
type Cache struct {
	Directory string
}

// Response is a minimal response object, which knows only about ListRecords and errors.
type Response struct {
	Date        string `xml:"responseDate"`
	ListRecords struct {
		Raw   string `xml:",innerxml"`
		Token struct {
			Value  string `xml:",chardata"`
			Cursor string `xml:"cursor,attr"`
			Size   string `xml:"completeListSize,attr"`
		} `xml:"resumptionToken"`
	} `xml:"ListRecords"`
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
		vals.AddIfExists("from", r.From.Format("2006-01-02"))
		vals.AddIfExists("until", r.Until.Format("2006-01-02"))
		vals.AddIfExists("metadataPrefix", r.Prefix)
		vals.AddIfExists("set", r.Set)
	} else {
		vals.Add("resumptionToken", r.ResumptionToken)
	}
	return fmt.Sprintf("%s?%s", r.Endpoint, vals.Encode())
}

// Do will execute one or more HTTP requests to fullfil this one OAI request.
// The record metadata is written verbatim to the given io.Writer.
func (req Request) Do(w io.Writer) error {
	for {
		if req.Verbose {
			log.Println(req.URL())
		}

		client := pester.New()
		client.MaxRetries = req.MaxRetry
		client.Backoff = pester.ExponentialBackoff

		resp, err := client.Get(req.URL())
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

// Filename returns the filename for a request, which only carries date
// information.
func (r CachedRequest) Filename() string {
	return fmt.Sprintf("%s-%s.xml", r.From.Format("2006-01-02"), r.Until.Format("2006-01-02"))
}

// Path returns the absolute path to the cache for an OAI request.
func (r CachedRequest) Path() string {
	u, _ := url.Parse(r.Endpoint)
	return path.Join(r.Cache.Directory, u.Host, r.Prefix, r.Fingerprint(), r.Filename())
}

// Do abstracts from the actual access (cache or HTTP). All OAI errors are
// returned back, except noRecordsMatch, which is used to indicate a zero
// result set.
func (r CachedRequest) Do(w io.Writer) error {
	if !r.IsCached() {
		file, err := ioutil.TempFile("", "oaimi-")
		if err != nil {
			return err
		}
		bw := bufio.NewWriter(file)
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
	if _, err := io.Copy(w, bufio.NewReader(file)); err != nil {
		return err
	}
	return nil
}

// Do runs a batched request over a range. All metadata gets written to the
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
				Verb:     "ListRecords",
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

// DateRange might be called TimeInterval as well.
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
