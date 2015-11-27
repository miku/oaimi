package next

import (
	"errors"
	"time"

	"github.com/jinzhu/now"
)

const oneDay = 24 * time.Hour

var ErrInvalidDateRange = errors.New("invalid date range")

// Window represent a span of time, from and until including.
type Window struct {
	From  time.Time
	Until time.Time
}

type TimeShiftFunc func(time.Time) time.Time

func (w Window) makeWindows(left, right TimeShiftFunc) ([]Window, error) {
	var ws []Window
	if w.From.After(w.Until) {
		return ws, ErrInvalidDateRange
	}
	var start, end time.Time
	from := w.From
	for {
		switch {
		case len(ws) == 0:
			start = now.New(w.From).BeginningOfDay()
		default:
			start = left(from)
		}
		end = right(from)
		if end.After(w.Until) {
			// discard end and use the end of day of until
			ws = append(ws, Window{From: start, Until: now.New(w.Until).EndOfDay()})
			break
		}
		ws = append(ws, Window{From: start, Until: end})
		from = end.Add(oneDay)
	}
	return ws, nil
}

func (w Window) Monthly() ([]Window, error) {
	shiftLeft := func(t time.Time) time.Time {
		return now.New(t).BeginningOfMonth()
	}
	shiftRight := func(t time.Time) time.Time {
		return now.New(t).EndOfMonth()
	}
	return w.makeWindows(shiftLeft, shiftRight)
}

func (w Window) Weekly() ([]Window, error) {
	shiftLeft := func(t time.Time) time.Time {
		return now.New(t).BeginningOfWeek()
	}
	shiftRight := func(t time.Time) time.Time {
		return now.New(t).EndOfWeek()
	}
	return w.makeWindows(shiftLeft, shiftRight)
}
