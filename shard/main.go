// Copyright (c) 2014 Eric Robert. All rights reserved.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"github.com/EricRobert/gometrics"
	"github.com/EricRobert/goshard"
	"log"
	"net/http"
	"os"
)

var (
	shards    = flag.Int("shards", 1, "number of shards")
	url       = flag.String("http-address", "", "<addr>:<port> to listen on for HTTP requests")
	reportUrl = flag.String("report-url", "", "URL where error reports are posted")
	metricUrl = flag.String("metric-url", "", "URL where metrics are posted")
	repeatUrl = flag.String("repeat-url", "", "URL where incoming requests are repeated as-is")
	routes    = flag.String("routes", "", "routes configuration")
)

type Route struct {
	Name    string
	Pattern string
	Kind    string
	Sharder json.RawMessage
}

type Routes []Route

func main() {
	flag.Parse()

	if *url == "" {
		log.Fatal("--http-address is required")
	}

	if *routes == "" {
		log.Fatal("--routes is required")
	}

	file, err := os.Open(*routes)
	if err != nil {
		log.Fatal(err.Error())
	}

	decoder := json.NewDecoder(file)
	s := Routes{}
	err = decoder.Decode(&s)
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, r := range s {
		e := shard.NewEndpoint(r.Name)
		t := shard.Table{
			Shards: *shards,
		}

		switch {
		case r.Kind == "json":
			js := shard.JsonSharder{
				Table: t,
			}

			e.Sharder = js

		default:
			log.Fatal("route doesn't specify a supported kind of endpoint")
		}

		err = json.Unmarshal(r.Sharder, &e.Sharder)
		if err != nil {
			log.Fatal(err.Error())
		}

		if reportUrl != nil {
			e.Reporter.SendFunc(func(value interface{}) {
				text, err := json.Marshal(value)
				if err != nil {
					panic(err.Error())
				}

				go http.Post(*reportUrl, "application/json", bytes.NewReader(text))
			})
		}

		if metricUrl != nil {
			e.Monitor.ReportFunc(func(s *metric.Summary) {
				text, err := json.Marshal(s)
				if err != nil {
					panic(err.Error())
				}

				go http.Post(*metricUrl, "application/json", bytes.NewReader(text))
			})
		}

		if repeatUrl != nil {
			e.RepeatUrl = *repeatUrl
		}

		e.Start()
		http.Handle(r.Pattern, e)
	}

	log.Printf("starting dispatcher at address=%s", *url)
	http.ListenAndServe(*url, nil)
}