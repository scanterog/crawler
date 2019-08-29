package crawler

import (
	"bytes"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrToURL(t *testing.T) {
	t.Run("Valid absolute URL", func(t *testing.T) {
		stringURL := "https://example.com"
		u, err := strToURL(stringURL)
		assert.NoError(t, err)
		assert.Equal(t, u.Scheme, "https")
		assert.Equal(t, u.Host, "example.com")
		assert.Equal(t, u.Path, "")
	})
	t.Run("Valid relative URL", func(t *testing.T) {
		stringURL := "/web"
		u, err := strToURL(stringURL)
		assert.NoError(t, err)
		assert.Equal(t, u.Path, "/web")
	})
	t.Run("Fragment trimmed from URL", func(t *testing.T) {
		stringURL := "/web#home"
		u, err := strToURL(stringURL)
		assert.NoError(t, err)
		assert.Equal(t, u.Path, "/web")
	})
	t.Run("Trailing slash trimmed", func(t *testing.T) {
		stringURL := "/home/"
		u, err := strToURL(stringURL)
		assert.NoError(t, err)
		assert.Equal(t, u.Path, "/home")
	})
	t.Run("Invalid scheme", func(t *testing.T) {
		stringURL := "ftp://example.com"
		u, err := strToURL(stringURL)
		assert.EqualError(t, err, ErrInvalidURLScheme.Error())
		assert.Nil(t, u)
	})
}

func TestStrToAbsoluteURL(t *testing.T) {
	t.Run("Valid absolute URL", func(t *testing.T) {
		stringURL := "https://example.com"
		u, err := strToAbsoluteURL(stringURL)
		assert.NoError(t, err)
		assert.Equal(t, u.Scheme, "https")
		assert.Equal(t, u.Host, "example.com")
		assert.Equal(t, u.Path, "")
	})
	t.Run("Relative URL not accepted", func(t *testing.T) {
		stringURL := "/web"
		u, err := strToAbsoluteURL(stringURL)
		assert.EqualError(t, err, ErrInvalidAbsoluteURL.Error())
		assert.Nil(t, u)
	})
	t.Run("Invalid absolute URL", func(t *testing.T) {
		stringURL := "example.com"
		u, err := strToAbsoluteURL(stringURL)
		assert.EqualError(t, err, ErrInvalidAbsoluteURL.Error())
		assert.Nil(t, u)
	})
}

func TestGetLinks(t *testing.T) {
	siteContent := []byte(`<!DOCTYPE html>
<html>
<head><title>Test page</title></head>
<body>
<a href="/home">home</a>
<a href="mailto:test@test.mock">
<a href="/help">help</a>
<a href="ftp://example.com">
<a href="https://twitter.com/"></a>
<a href="/">/</a>
</body>
</html>`)
	u, err := getLinks(bytes.NewReader(siteContent))
	assert.NoError(t, err)
	assert.Equal(t, u[0], "/home")
	assert.Equal(t, u[1], "mailto:test@test.mock")
	assert.Equal(t, u[2], "/help")
	assert.Equal(t, u[3], "ftp://example.com")
	assert.Equal(t, u[4], "https://twitter.com/")
	assert.Equal(t, u[5], "/")
}

func TestIsExternalURL(t *testing.T) {
	t.Run("External URL", func(t *testing.T) {
		site := webSite{
			URL:    &url.URL{Host: "twitter.com"},
			Parent: &url.URL{Host: "example.com"},
		}
		assert.True(t, isExternalURL(site))
	})
	t.Run("No external URL", func(t *testing.T) {
		site := webSite{
			URL:    &url.URL{Host: "example.com"},
			Parent: &url.URL{Host: "example.com"},
		}
		assert.False(t, isExternalURL(site))
	})
}

func TestIsRelativeURL(t *testing.T) {
	t.Run("Relative URL", func(t *testing.T) {
		u := &url.URL{Path: "/"}
		assert.True(t, isRelativeURL(u))
	})
	t.Run("No relative URL", func(t *testing.T) {
		u := &url.URL{Scheme: "https", Host: "example.com", Path: "/"}
		assert.False(t, isRelativeURL(u))
	})
}

func TestIsMediaURL(t *testing.T) {
	t.Run("Media URL", func(t *testing.T) {
		u := &url.URL{Path: "/file.pdf"}
		assert.True(t, isMediaURL(u))
	})
	t.Run("No media URL", func(t *testing.T) {
		u := &url.URL{Scheme: "https", Host: "example.com", Path: "/about"}
		assert.False(t, isMediaURL(u))
	})
}
