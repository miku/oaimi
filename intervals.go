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
