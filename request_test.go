package oaimi

import (
	"testing"
	"time"
)

func TestRequestURL(t *testing.T) {
	var tests = []struct {
		req Request
		url string
		err error
	}{
		{Request{}, "", ErrNoEndpoint},
		{Request{Endpoint: "Hello"}, "", ErrNoVerb},
		{Request{Endpoint: "Hello", Verb: "x"}, "", ErrBadVerb},
		{Request{Endpoint: "Hello", Verb: "Identify"}, "Hello?verb=Identify", nil},
		{Request{Endpoint: "http://example.com/oai", Verb: "Identify"}, "http://example.com/oai?verb=Identify", nil},
		{Request{Endpoint: "http://example.com/oai",
			Verb: "Identify",
			From: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		}, "http://example.com/oai?verb=Identify", nil},
		{Request{Endpoint: "http://example.com/oai", Verb: "ListRecords",
			From:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Until: time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC)},
			"http://example.com/oai?from=2000-01-01&until=2000-01-02&verb=ListRecords", nil},
		{Request{Endpoint: "http://example.com/oai", Verb: "ListRecords",
			From:            time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
			Until:           time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
			ResumptionToken: "1"},
			"http://example.com/oai?resumptionToken=1&verb=ListRecords", nil},
		{Request{Endpoint: "http://example.com/oai",
			Verb: "ListRecords", Set: "X"}, "http://example.com/oai?set=X&verb=ListRecords", nil},
		{Request{Endpoint: "http://example.com/oai",
			Verb: "ListRecords", Set: "X", Prefix: "P"}, "http://example.com/oai?metadataPrefix=P&set=X&verb=ListRecords", nil},
		{Request{Endpoint: "http://example.com/oai",
			Verb: "ListRecords", Set: "X", Prefix: "P", ResumptionToken: "R"},
			"http://example.com/oai?resumptionToken=R&verb=ListRecords", nil},
	}

	for _, test := range tests {
		got, err := test.req.URL()
		if err != test.err {
			t.Errorf("r.URL() got %v, want %v", err, test.err)
		}
		if got != test.url {
			t.Errorf("r.URL() got %v, want %v", got, test.url)
		}
	}
}

func TestMakeCachePath(t *testing.T) {
	var tests = []struct {
		req Request
		p   string
		err error
	}{
		{
			Request{
				Verb:     "ListRecords",
				Endpoint: "http://www.doabooks.org/oai",
				From:     time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
				Until:    time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC),
				Prefix:   "marcxml"},
			"www.doabooks.org/oai/ListRecords/marcxml/2000-01-01-2001-01-01.xml",
			nil,
		},
		{
			Request{
				Verb:     "ListRecords",
				Endpoint: "http://www.doabooks.org/oai",
				Prefix:   "marcxml"},
			"",
			ErrMissingFromOrUntil,
		},
		{
			Request{
				Verb:     "ListRecords",
				Endpoint: "",
				Prefix:   "marcxml"},
			"",
			ErrNoHost,
		},
	}
	for _, test := range tests {
		result, err := makeCachePath(test.req)
		if err != test.err {
			t.Errorf("makeCachePath(), got %v, want %v", err, test.err)
		}
		if result != test.p {
			t.Errorf("makeCachePath(), got %v, want %v", result, test.p)
		}
	}
}
