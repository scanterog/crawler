package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/scanterog/crawler/crawler"
	log "github.com/sirupsen/logrus"
)

var (
	helpMsgNumWorkers        = "Number of concurrent workers crawling sites."
	helpMsgHttpClientTimeout = "Time limit (in sec) for a HTTP request. A Timeout of zero means no timeout."
	helpMsgSiteMapOutputFile = "File path where the site map will be written to."
	helpMsgDebug             = "Enable debug mode."
)

func main() {
	numWorkers := flag.Int("num-workers", crawler.DefaultNumWorkers, helpMsgNumWorkers)
	httpClientTimeout := flag.Int("client-timeout", crawler.DefaultHTTPClientTimeoutSec, helpMsgHttpClientTimeout)
	siteMapOutputFile := flag.String("output-file", os.Stdout.Name(), helpMsgSiteMapOutputFile)
	debug := flag.Bool("debug", false, helpMsgDebug)
	flag.Parse()

	log.SetLevel(log.InfoLevel)
	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	args := flag.Args()
	if len(args) != 1 {
		usage()
	}
	seedURL := args[0]

	c := crawler.Crawler{
		SeedURL:              seedURL,
		NumWorkers:           *numWorkers,
		HTTPClientTimeoutSec: *httpClientTimeout,
		SiteMapOutputFile:    *siteMapOutputFile,
	}

	start := time.Now()
	err := c.Run()
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Crawling took %v", time.Since(start))
}

func usage() {
	const msg string = "Usage: %s [flags] SEED_URL\n"
	fmt.Fprintf(os.Stderr, msg, os.Args[0])
	fmt.Fprintf(os.Stderr, "\nFlags:\n")
	flag.PrintDefaults()
	os.Exit(1)
}
