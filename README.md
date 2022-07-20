# go-sundheit-opentelemetry


[![Actions Status](https://github.com/AppsFlyer/go-sundheit-opentelemetry/workflows/go-build/badge.svg)](https://github.com/AppsFlyer/go-sundheit-opentelemetry/actions)
[![CircleCI](https://circleci.com/gh/AppsFlyer/go-sundheit-opentelemetry.svg?style=svg)](https://circleci.com/gh/AppsFlyer/go-sundheit-opentelemetry)
[![Coverage Status](https://coveralls.io/repos/github/AppsFlyer/go-sundheit-opentelemetry/badge.svg?branch=master)](https://coveralls.io/github/AppsFlyer/go-sundheit-opentelemetry?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/AppsFlyer/go-sundheit-opentelemetry)](https://goreportcard.com/report/github.com/AppsFlyer/go-sundheit-opentelemetry)
[![Godocs](https://img.shields.io/badge/golang-documentation-blue.svg)](https://godoc.org/github.com/AppsFlyer/go-sundheit-opentelemetry)

<img align="right" src="docs/go-sundheit.png" width="200">

A library built to provide support for open telemetry metrics for go-sundheit.

## Installation
Using go modules:
```
go get github.com/AppsFlyer/go-sundheit-opentelemetry@v0.0.1
```

## Usage
```go
import (
	"net/http"
	"time"
	"log"

	"github.com/pkg/errors"
	"github.com/AppsFlyer/go-sundheit"
    sundheit_opentelemetry "github.com/AppsFlyer/go-sundheit-opentelemetry"

    healthhttp "github.com/AppsFlyer/go-sundheit/http"
	"github.com/AppsFlyer/go-sundheit/checks"
)

func main() {
    // creates otel metrics listener
    ot := sundheit_opentelemetry.NewMetricsListener()

	// create a new health instance
	h := gosundheit.New(gosundheit.WithCheckListeners(ot), gosundheit.WithHealthListeners(ot))
	
	// define an HTTP dependency check
	httpCheckConf := checks.HTTPCheckConfig{
		CheckName: "httpbin.url.check",
		Timeout:   1 * time.Second,
		// dependency you're checking - use your own URL here...
		// this URL will fail 50% of the times
		URL:       "http://httpbin.org/status/200,300",
	}
	// create the HTTP check for the dependency
	// fail fast when you misconfigured the URL. Don't ignore errors!!!
	httpCheck, err := checks.NewHTTPCheck(httpCheckConf)
	if err != nil {
		fmt.Println(err)
		return // your call...
	}

	// Alternatively panic when creating a check fails
	httpCheck = checks.Must(checks.NewHTTPCheck(httpCheckConf))

	err = h.RegisterCheck(
		httpCheck,
		gosundheit.InitialDelay(time.Second),         // the check will run once after 1 sec
		gosundheit.ExecutionPeriod(10 * time.Second), // the check will be executed every 10 sec
	)
	
	if err != nil {
		fmt.Println("Failed to register check: ", err)
		return // or whatever
	}

	// define more checks...
	
	// register a health endpoint
	http.Handle("/admin/health.json", healthhttp.HandleHealthJSON(h))
	
	// serve HTTP
	log.Fatal(http.ListenAndServe(":8080", nil))
}
```
