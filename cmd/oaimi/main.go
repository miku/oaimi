package main

import (
	"flag"
	"log"
	"time"
)

func BeginningOfDay(now time.Time) time.Time {
	d := time.Duration(-now.Hour()) * time.Hour
	return now.Truncate(time.Hour).Add(d)
}

func DateRangeDaily(s, t time.Time) []time.Time {
	var dates []time.Time
	b := BeginningOfDay(s)
	for b.Before(t) {
		dates = append(dates, b)
		b = b.Add(24 * time.Hour)
	}
	return dates
}

func main() {
	set := flag.String("set", "", "OAI set")
	prefix := flag.String("prefix", "oai_dc", "OAI metadataPrefix")
	from := flag.String("from", "1970-01-01", "OAI from")
	until := flag.String("until", time.Now().Format("2006-01-02"), "OAI until")

	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatal("URL to OAI endpoint required")
	}

	From, err := time.Parse("2006-01-02", *from)
	if err != nil {
		log.Fatal(err)
	}
	Until, err := time.Parse("2006-01-02", *until)
	if err != nil {
		log.Fatal(err)
	}

	if Until.Before(From) {
		log.Fatal("invalid date range")
	}

	endpoint := flag.Arg(0)
	log.Printf("%s?verb=ListRecords&metadataPrefix=%s&from=%s&until=%s&set=%s\n", endpoint, *prefix,
		From.Format("2006-01-02"), Until.Format("2006-01-02"), *set)
}
