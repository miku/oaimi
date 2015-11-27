package next

import "testing"

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
			Verb: "Identify", From: "123"}, "http://example.com/oai?from=123&verb=Identify", nil},
		{Request{Endpoint: "http://example.com/oai",
			Verb: "ListRecords", From: "123", Until: "456"}, "http://example.com/oai?from=123&until=456&verb=ListRecords", nil},
		{Request{Endpoint: "http://example.com/oai",
			Verb: "ListRecords", From: "123", Until: "456", ResumptionToken: "1"},
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
