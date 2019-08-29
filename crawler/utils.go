package crawler

import (
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// strToURL parses a string and returns an url.URL object.
// The URL might be absolute or relative.
// Only http(s) schemes are considered valid.
// Fragments are ignored and trailing slashes are removed.
func strToURL(stringUrl string) (*url.URL, error) {
	u, err := url.Parse(stringUrl)
	if err != nil {
		return nil, ErrInvalidURL
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return nil, ErrInvalidURLScheme
	}
	u.Fragment = ""
	u.Path = strings.TrimSuffix(u.Path, "/")
	if u.Path == "." {
		u.Path = ""
	}
	return u, nil
}

// strToAbsoluteURL parses a string and returns an url.URL object
// only if it is an absolute URL (full URL).
func strToAbsoluteURL(stringUrl string) (*url.URL, error) {
	u, err := strToURL(stringUrl)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, ErrInvalidAbsoluteURL
	}
	return u, nil
}

// getLinks parses the HTML document and returns a list
// of URLs as a list of strings.
func getLinks(siteContent io.Reader) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(siteContent)
	if err != nil {
		return nil, err
	}

	var links []string
	doc.Find("a").Each(func(index int, element *goquery.Selection) {
		href, exists := element.Attr("href")
		if exists {
			links = append(links, href)
		}
	})

	return links, nil
}

func isExternalURL(s webSite) bool {
	return s.Parent != nil && s.URL.Host != s.Parent.Host
}

func isRelativeURL(u *url.URL) bool {
	return u.Scheme == "" || u.Host == ""
}

func isMediaURL(u *url.URL) bool {
	r, _ := regexp.Compile(`\.(jpg|jpeg|png|svg|gif|pdf|csv)$`)
	return r.MatchString(u.Path)
}
