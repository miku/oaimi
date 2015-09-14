package oaimi

import (
	"bufio"
	"crypto/sha1"
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

	"github.com/jinzhu/now"
)

const Version = "0.1.0"

var (
	ErrInvalidDateRange = errors.New("invalid date range")
	ErrRetriesExceeded  = errors.New("retried and failed too many times")

	backoff = 50
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
	MaxRetry        uint
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
		var attempt uint
		var decoder *xml.Decoder
		var response Response

		for {
			if attempt > req.MaxRetry {
				return ErrRetriesExceeded
			}
			if req.Verbose {
				log.Printf("%d %s", attempt, req.URL())
			}
			resp, err := http.Get(req.URL())
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			decoder = xml.NewDecoder(resp.Body)
			decoder.Decode(&response)

			if resp.StatusCode < 400 {
				break
			}
			if req.Verbose {
				log.Println("got %s, retrying...", resp.Status)
			}
			attempt++
			pause := time.Duration(2 << attempt * backoff)
			time.Sleep(pause * time.Millisecond)
		}

		if response.Error.Code != "" {
			return OAIError{Code: response.Error.Code, Message: response.Error.Message}
		}

		_, err := w.Write([]byte(response.ListRecords.Raw))
		if err != nil {
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

// Path returns the absolute path to the cache for an OAI request.
func (r CachedRequest) Path() string {
	h := sha1.New()
	io.WriteString(h, fmt.Sprintf("%s:%s:%s", r.Endpoint, r.Set, r.Prefix))
	fn := fmt.Sprintf("%s-%s.xml", r.From.Format("2006-01-02"), r.Until.Format("2006-01-02"))
	return path.Join(r.Cache.Directory, fmt.Sprintf("%x", h.Sum(nil)), fn)
}

// Do might abstract from the actual access (cache or HTTP).
func (r CachedRequest) Do(w io.Writer) error {
	if !r.IsCached() {
		// store reply in temporary place for atomicity
		file, err := ioutil.TempFile("", "oaimi-")
		if err != nil {
			return err
		}

		bw := bufio.NewWriter(file)

		err = r.Request.Do(bw)
		if err != nil {
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

		dirname := filepath.Dir(r.Path())
		if _, err := os.Stat(dirname); os.IsNotExist(err) {
			err := os.MkdirAll(dirname, 0755)
			if err != nil {
				return err
			}
		}

		err = os.Rename(file.Name(), r.Path())
		if err != nil {
			return err
		}
	}
	file, err := os.Open(r.Path())
	if err != nil {
		return err
	}

	_, err = io.Copy(w, bufio.NewReader(file))
	if err != nil {
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
		err := req.Do(w)
		if err != nil {
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
