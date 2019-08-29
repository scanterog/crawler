package crawler_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/scanterog/crawler/crawler"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInitialConfigValidation(t *testing.T) {
	t.Run("Invalid absolute SeedURL", func(t *testing.T) {
		c := crawler.Crawler{
			SeedURL: "/home",
		}
		err := c.Run()
		assert.Error(t, err, crawler.ErrInvalidAbsoluteURL)
	})

	t.Run("Invalid SeedURL scheme", func(t *testing.T) {
		c := crawler.Crawler{
			SeedURL: "ftp://example.com",
		}
		err := c.Run()
		assert.Error(t, err, crawler.ErrInvalidURLScheme)
	})

	t.Run("Invalid number workers", func(t *testing.T) {
		c := crawler.Crawler{
			NumWorkers: 0,
		}
		err := c.Run()
		assert.Error(t, err, crawler.ErrInvalidNumWorkers)
	})

	t.Run("Invalid httpClientTimeout", func(t *testing.T) {
		c := crawler.Crawler{
			HTTPClientTimeoutSec: -1,
		}
		err := c.Run()
		assert.Error(t, err, crawler.ErrInvalidHTTPClientTimeout)
	})

	t.Run("Invalid SiteMapOutputFile", func(t *testing.T) {
		c := crawler.Crawler{
			SiteMapOutputFile: "/tmp/",
		}
		err := c.Run()
		assert.Error(t, err)
	})
}

func TestRun(t *testing.T) {
	httpTestServer := newTestServer()
	defer httpTestServer.Close()

	siteMapOutBuf := &bytes.Buffer{}
	c := crawler.Crawler{
		SeedURL:              httpTestServer.URL,
		NumWorkers:           crawler.DefaultNumWorkers,
		HTTPClientTimeoutSec: crawler.DefaultHTTPClientTimeoutSec,
		SiteMapWriter:        siteMapOutBuf,
	}
	errBuf := &bytes.Buffer{}
	log.SetOutput(errBuf)

	err := c.Run()
	assert.NoError(t, err)

	errLog := errBuf.String()
	assert.True(t, strings.Contains(errLog, "404 Not Found"))
	outSiteMapTokens := strings.Split(siteMapOutBuf.String(), "\n")
	generatedSiteMap := []string{}
	for _, line := range outSiteMapTokens[:len(outSiteMapTokens)-1] {
		generatedSiteMap = append(generatedSiteMap, fmt.Sprintf("%s\n", line))
	}
	expectedSiteMap := getExpectedSiteMap(httpTestServer.URL)
	// siteMap might be in any order on units of "result"
	assert.ElementsMatch(t, generatedSiteMap, expectedSiteMap)
}

// Helpers
func getExpectedSiteMap(serverURL string) []string {
	mainPage := fmt.Sprintf("%s", serverURL)
	aboutPage := fmt.Sprintf("%s/about", serverURL)
	helpPage := fmt.Sprintf("%s/help", serverURL)
	careersPage := fmt.Sprintf("%s/careers", serverURL)

	siteMap := map[string][]string{
		mainPage:    []string{aboutPage, helpPage, "https://twitter.com"},
		aboutPage:   []string{mainPage, helpPage, "https://fb.com", careersPage},
		careersPage: []string{mainPage, "https://golang.org", helpPage},
	}

	siteMapList := []string{}
	for parent, links := range siteMap {
		for _, link := range links {
			siteMapList = append(siteMapList, fmt.Sprintf("%v -> %v\n", parent, link))
		}
	}
	return siteMapList
}

func newTestServer() *httptest.Server {
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Main page</title></head>
<body>
<a href="/about">about</a>
<a href="mailto:test@test.mock">
<a href="/help">help</a>
<a href="https://twitter.com/"></a>
</body>
</html>
		`))
	})

	httpMux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>About page</title></head>
<body>
<a href="/">Main</a>
<a href="/help">help</a>
<a href="https://fb.com"></a>
<a href="/careers">careers</a>
</body>
</html>
		`))
	})

	httpMux.HandleFunc("/careers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Careers page</title></head>
<body>
<a href="/">Main</a>
<a href="https://golang.org/">Learn go</a>
<a href="/help">help</a>
</body>
</html>
		`))
	})

	httpMux.HandleFunc("/help", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	return httptest.NewServer(httpMux)
}
