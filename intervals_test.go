package oaimi

import (
	"reflect"
	"testing"
	"time"
)

func TestWindowMonthly(t *testing.T) {
	var tests = []struct {
		w  Window
		ws []Window
	}{
		{
			w: Window{From: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Until: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
			ws: []Window{
				Window{
					From:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.UTC),
				},
			},
		},
		{
			w: Window{From: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Until: time.Date(2000, 5, 1, 0, 0, 0, 0, time.UTC)},
			ws: []Window{
				Window{
					From:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 1, 31, 23, 59, 59, 999999999, time.UTC),
				},
				Window{
					From:  time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 2, 29, 23, 59, 59, 999999999, time.UTC),
				},
				Window{
					From:  time.Date(2000, 3, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 3, 31, 23, 59, 59, 999999999, time.UTC),
				},
				Window{
					From:  time.Date(2000, 4, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 4, 30, 23, 59, 59, 999999999, time.UTC),
				},
				Window{
					From:  time.Date(2000, 5, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 5, 1, 23, 59, 59, 999999999, time.UTC),
				},
			},
		},
		{
			w: Window{From: time.Date(2001, 12, 11, 9, 0, 0, 0, time.UTC), Until: time.Date(2002, 1, 16, 12, 0, 0, 0, time.UTC)},
			ws: []Window{
				Window{
					From:  time.Date(2001, 12, 11, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2001, 12, 31, 23, 59, 59, 999999999, time.UTC),
				},
				Window{
					From:  time.Date(2002, 1, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2002, 1, 16, 23, 59, 59, 999999999, time.UTC),
				},
			},
		},
	}

	for _, test := range tests {
		result := test.w.Monthly()
		if !reflect.DeepEqual(result, test.ws) {
			t.Errorf("Monthly() got %v, want %v", result, test.ws)
		}
	}
}

func TestWindowWeekly(t *testing.T) {
	var tests = []struct {
		w  Window
		ws []Window
	}{
		{
			w: Window{From: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Until: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
			ws: []Window{
				Window{
					From:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.UTC),
				},
			},
		},
		{
			w: Window{From: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), Until: time.Date(2000, 2, 1, 0, 0, 0, 0, time.UTC)},
			ws: []Window{
				Window{
					From:  time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 1, 1, 23, 59, 59, 999999999, time.UTC),
				},
				Window{
					From:  time.Date(2000, 1, 2, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 1, 8, 23, 59, 59, 999999999, time.UTC),
				},
				Window{
					From:  time.Date(2000, 1, 9, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 1, 15, 23, 59, 59, 999999999, time.UTC),
				},
				Window{
					From:  time.Date(2000, 1, 16, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 1, 22, 23, 59, 59, 999999999, time.UTC),
				},
				Window{
					From:  time.Date(2000, 1, 23, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 1, 29, 23, 59, 59, 999999999, time.UTC),
				},
				Window{
					From:  time.Date(2000, 1, 30, 0, 0, 0, 0, time.UTC),
					Until: time.Date(2000, 2, 1, 23, 59, 59, 999999999, time.UTC),
				},
			},
		},
	}

	for _, test := range tests {
		result := test.w.Weekly()
		if !reflect.DeepEqual(result, test.ws) {
			t.Errorf("Weekly() got %v, want %v", result, test.ws)
		}
	}
}
