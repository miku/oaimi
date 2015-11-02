package oaimi

import (
	"reflect"
	"testing"
	"time"
)

func MustParse(layout, s string) time.Time {
	t, err := time.Parse(layout, s)
	if err != nil {
		panic(err)
	}
	return t
}

func MustParseDefault(s string) time.Time {
	return MustParse("2006-01-02", s)
}

func TestRequestPath(t *testing.T) {
	var cases = []struct {
		Request CachedRequest
		Name    string
	}{
		{
			Request: CachedRequest{
				Cache: Cache{
					Directory: "/",
				},
				Request: Request{
					Endpoint: "https://abc.xyz",
					Set:      "abc://1/@18",
				},
			},
			Name: "/abc.xyz/aHR0cHM6Ly9hYmMueHl6I2FiYzovLzEvQDE4/0001-01-01-0001-01-01.xml.gz",
		},
		{
			Request: CachedRequest{
				Cache: Cache{
					Directory: "/",
				},
				Request: Request{
					Endpoint: "https://abc.xyz",
					Set:      "abc://1/@18",
					Prefix:   "marc21",
				},
			},
			Name: "/abc.xyz/marc21/aHR0cHM6Ly9hYmMueHl6I2FiYzovLzEvQDE4/0001-01-01-0001-01-01.xml.gz",
		},
		{
			Request: CachedRequest{
				Cache: Cache{
					Directory: "/",
				},
				Request: Request{
					Endpoint: "https://abc.xyz",
					Set:      "abc://1/@18",
					Prefix:   "marc22",
					From:     MustParseDefault("2010-01-01"),
					Until:    MustParseDefault("2015-01-01"),
				},
			},
			Name: "/abc.xyz/marc22/aHR0cHM6Ly9hYmMueHl6I2FiYzovLzEvQDE4/2010-01-01-2015-01-01.xml.gz",
		},
	}

	for _, c := range cases {
		if c.Request.Path() != c.Name {
			t.Errorf("Request.Name got %s, want %s", c.Request.Path(), c.Name)
		}
	}
}

func TestMonthlyDateRange(t *testing.T) {
	var cases = []struct {
		From   time.Time
		Until  time.Time
		Ranges []DateRange
		err    error
	}{
		{
			From:  MustParseDefault("2010-01-01"),
			Until: MustParseDefault("2010-02-01"),
			Ranges: []DateRange{
				DateRange{
					From:  time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2010, 1, 31, 23, 59, 59, 999999999, time.UTC),
				},
				DateRange{
					From:  time.Date(2010, 2, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2010, 2, 1, 23, 59, 59, 999999999, time.UTC),
				},
			},
			err: nil,
		},
		{
			From:  MustParseDefault("2010-01-01"),
			Until: MustParseDefault("2010-03-02"),
			Ranges: []DateRange{
				DateRange{
					From:  time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2010, 1, 31, 23, 59, 59, 999999999, time.UTC),
				},
				DateRange{
					From:  time.Date(2010, 2, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2010, 2, 28, 23, 59, 59, 999999999, time.UTC),
				},
				DateRange{
					From:  time.Date(2010, 3, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2010, 3, 2, 23, 59, 59, 999999999, time.UTC),
				},
			},
			err: nil,
		},
		{
			From:  MustParseDefault("2010-01-10"),
			Until: MustParseDefault("2010-03-02"),
			Ranges: []DateRange{
				DateRange{
					From:  time.Date(2010, 1, 10, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2010, 1, 31, 23, 59, 59, 999999999, time.UTC),
				},
				DateRange{
					From:  time.Date(2010, 2, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2010, 2, 28, 23, 59, 59, 999999999, time.UTC),
				},
				DateRange{
					From:  time.Date(2010, 3, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2010, 3, 2, 23, 59, 59, 999999999, time.UTC),
				},
			},
			err: nil,
		},
		{
			From:  MustParseDefault("2010-01-10"),
			Until: MustParseDefault("2010-01-19"),
			Ranges: []DateRange{
				DateRange{
					From:  time.Date(2010, 1, 10, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2010, 1, 19, 23, 59, 59, 999999999, time.UTC),
				},
			},
			err: nil,
		},
		{
			From:  MustParseDefault("2010-04-01"),
			Until: MustParseDefault("2010-03-02"),
			err:   ErrInvalidDateRange,
		},
	}

	for _, c := range cases {
		ranges, err := MonthlyDateRange(c.From, c.Until)
		if err != c.err {
			t.Errorf("MonthlyDateRange got %v, want %v", err, c.err)
		}
		if !reflect.DeepEqual(ranges, c.Ranges) {
			t.Errorf("MonthlyDateRange got %#v, want %#v", ranges, c.Ranges)
		}
	}
}
