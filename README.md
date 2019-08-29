# Crawler

This provides a concurrent crawling execution limited to a single subdomain (no external URLs are followed) in order to produce a simple textual sitemap. The main idea was to exercise concurrency in Go.

## Install

```
go get github.com/scanterog/crawler
```

## Usage

```
crawler https://gobyexample.com
```

To redirect output to a file:
```
crawler -output-file /tmp/gobyexample.com https://gobyexample.com
```

## Limitations

* Only one seed URL. It does not accept a list of initial URLs.
* One subdomain. If we start with https://wikipedia.org, it will crawl all pages within wikipedia.org but not follow external links. For example facebook.com or uk.wikipedia.org.
* No politeness mechanism supported like robots.txt.