// Package crawler provides a concurrent crawling execution
// limited to a single subdomain (no external URLs are followed)
// in order to produce a textual sitemap
package crawler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	DefaultNumWorkers           = 5
	DefaultHTTPClientTimeoutSec = 2
	DefaultCrawlerUserAgent     = "CrawlerBot/0.1"
)

var (
	ErrInvalidURL               = errors.New("invalid URL")
	ErrInvalidAbsoluteURL       = errors.New("invalid absolute URL")
	ErrInvalidURLScheme         = errors.New("invalid URL scheme: only http(s) supported")
	ErrInvalidNumWorkers        = errors.New("invalid number of workers")
	ErrInvalidHTTPClientTimeout = errors.New("invalid HTTP Client timeout: it must be at least 0 (no timeout)")
)

type Crawler struct {
	SeedURL              string          // initial str URL for crawling
	NumWorkers           int             // number of concurrent workers polling the job queue
	HTTPClientTimeoutSec int             // time limit (in seconds) for a HTTP request
	SiteMapOutputFile    string          // file where the site map will be written to
	SiteMapWriter        io.Writer       // this takes precedence over SiteMapOutputFile. If this is not provided, this is backed by SiteMapOutputFile.
	workQueue            chan webSite    // job queue - collection of WebSites - for the workers
	workQueueCapacity    int             // max numbers of elements before the write to the queue gets blocked
	workQueueDelta       chan int        // channel for notifying enqueue/dequeue operations
	siteFilterQueue      chan webSite    // intermediate channel for filtering before adding more WebSites to the queue
	visitedSites         map[string]bool // set for keeping the collection of already visited sites
	resultQueue          chan result     // channel for sending the scrape result
	siteMapDone          chan bool       // channel for signaling the end of the site map build
	wg                   sync.WaitGroup  // waitGroup for waiting on workers to finish execution
	startOnce            sync.Once       // avoid executing init more than once.
}

type webSite struct {
	URL    *url.URL
	Parent *url.URL
}

type result struct {
	SourceSite    webSite
	ChildrenSites []*webSite
}

// Run runs the crawling process by spawning "NumWorkers" workers and
// performing the scraping in each site found. It generates the textual
// site map in the provided "SiteMapOutputFile" file.
func (c *Crawler) Run() error {
	err := c.validate()
	if err != nil {
		return err
	}
	c.startOnce.Do(c.init)

	log.Debug("Crawler started")
	go func() {
		u, _ := strToAbsoluteURL(c.SeedURL)
		c.siteFilterQueue <- webSite{URL: u, Parent: nil}
		c.workQueueDelta <- 1
	}()

	go c.workQueueDoneChecker()
	go c.workQueueAppender()
	go c.siteMapBuilder()

	for i := 0; i < c.NumWorkers; i++ {
		c.wg.Add(1)
		go c.startWorker(i)
	}
	c.wg.Wait()
	close(c.siteFilterQueue)
	close(c.resultQueue)
	<-c.siteMapDone
	return nil
}

func (c *Crawler) validate() error {
	_, err := strToAbsoluteURL(c.SeedURL)
	if err != nil {
		return err
	}
	if c.NumWorkers <= 0 {
		return ErrInvalidNumWorkers
	}
	if c.HTTPClientTimeoutSec < 0 {
		return ErrInvalidHTTPClientTimeout
	}
	if c.SiteMapOutputFile == "" {
		c.SiteMapOutputFile = os.Stdout.Name()
	}
	if c.SiteMapWriter == nil {
		f, err := os.Create(c.SiteMapOutputFile)
		if err != nil {
			return fmt.Errorf("can't create siteMap output file: %q", err.Error())
		}
		c.SiteMapWriter = f
	}
	return nil
}

func (c *Crawler) init() {
	c.workQueueCapacity = c.NumWorkers * 2
	c.workQueue = make(chan webSite, c.workQueueCapacity)
	c.workQueueDelta = make(chan int)
	c.siteFilterQueue = make(chan webSite, c.workQueueCapacity)
	c.visitedSites = make(map[string]bool)
	c.resultQueue = make(chan result, c.workQueueCapacity)
	c.siteMapDone = make(chan bool)
}

func (c *Crawler) workQueueDoneChecker() {
	log.Debug("workQueueDoneChecker started.")
	workQueueSize := 0
	for delta := range c.workQueueDelta {
		workQueueSize += delta
		log.Debugf("Current WorkQueueSize is: %d", workQueueSize)
		if workQueueSize == 0 {
			close(c.workQueue)
		}
	}
}

func (c *Crawler) workQueueAppender() {
	log.Debug("workQueueAppender started.")
	for newSite := range c.siteFilterQueue {
		if isExternalURL(newSite) || isMediaURL(newSite.URL) {
			c.workQueueDelta <- -1
			continue

		}

		siteURL := strings.TrimPrefix(newSite.URL.String(), newSite.URL.Scheme)
		if !c.visitedSites[siteURL] {
			c.visitedSites[siteURL] = true
			c.workQueue <- newSite
		} else {
			c.workQueueDelta <- -1
		}
	}
}

func (c *Crawler) siteMapBuilder() {
	for r := range c.resultQueue {
		for _, s := range r.ChildrenSites {
			line := fmt.Sprintf("%v -> %v\n", r.SourceSite.URL.String(), s.URL.String())
			fmt.Fprint(c.SiteMapWriter, line)
		}
	}
	c.siteMapDone <- true
}

func (c *Crawler) startWorker(id int) {
	log.Debugf("Started worker %d", id)
	defer c.wg.Done()
	for site := range c.workQueue {
		log.Debugf("[worker %d] Reading site out of work queue: %v\n", id, site)
		newSites, err := c.scrape(site)
		if err != nil {
			log.Errorf("Failed to parse %q: %s", site.URL.String(), err.Error())
			c.workQueueDelta <- -1
			continue
		}

		c.resultQueue <- result{SourceSite: site, ChildrenSites: newSites}

		go func() {
			for _, s := range newSites {
				c.siteFilterQueue <- *s
			}
		}()

		c.workQueueDelta <- -1
	}
}

func (c *Crawler) scrape(s webSite) ([]*webSite, error) {
	log.Debugf("Starting to parse webSite: %v", s)
	client := &http.Client{
		Timeout: time.Duration(c.HTTPClientTimeoutSec) * time.Second,
	}

	request, err := http.NewRequest("GET", s.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", DefaultCrawlerUserAgent)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("%v", response.Status)
	}

	sites, err := s.getNewSites(response.Body)
	if err != nil {
		return nil, err
	}

	if len(sites) != 0 {
		c.workQueueDelta <- len(sites)
	}

	return sites, nil
}

func (s webSite) getNewSites(siteContent io.Reader) ([]*webSite, error) {
	log.Debugf("Starting to get new webSites for %v", s)
	links, err := getLinks(siteContent)
	if err != nil {
		return nil, fmt.Errorf("failed to get links: %s", err.Error())
	}
	log.Debugf("Extracted links: %v", links)

	urlSet := map[string]bool{}
	var newSites []*webSite
	for _, link := range links {
		newURL, err := strToURL(link)
		if err != nil {
			log.Debugf("Skipping %q. Error: %q", link, err.Error())
			continue
		}

		if isRelativeURL(newURL) {
			if newURL.Scheme == "" {
				newURL.Scheme = s.URL.Scheme
			}
			if newURL.Host == "" {
				newURL.Host = s.URL.Host
			}
		}

		if !urlSet[newURL.String()] {
			urlSet[newURL.String()] = true
			log.Debugf("Appending newSite: %s -> %s", s.URL.String(), newURL.String())
			newSites = append(newSites, &webSite{URL: newURL, Parent: s.URL})
		}
	}

	return newSites, nil
}
