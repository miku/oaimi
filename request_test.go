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
