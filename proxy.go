// Package header_pattern_proxy
package multiplexer_proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
        "github.com/patrickmn/go-cache"
	"time"
)

// Config the plugin configuration.
type Config struct {
	Header  string            `json:"header,omitempty"` // target header
        TargetMatch string        `json:"target_match,omitempty"`
        TargetReplace string      `json:"target_replace,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Header:  "",
		TargetMatch: "^(.*)$",
		TargetReplace: "${header}.$1"
	}
}

type SiteProxy struct {
	config *Config
        proxyCache  *cache
        pattern1 *Regexp
        pattern2 *Regexp
	next   http.Handler
	name   string
}

// New created a new SiteProxy plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.Header) == 0 {
		return nil, fmt.Errorf("header cannot be empty")
	}

	if len(config.TargetMatch) == 0 {
		return nil, fmt.Errorf("target_match cannot be empty")
	}
	if len(config.TargetReplace) == 0 {
		return nil, fmt.Errorf("target_replace cannot be empty")
	}

	return &SiteProxy{
		config: config,
                cache: cache.New(5*time.Minute, 10*time.Minute),
		pattern1 : regexp.MustCompile(`\${header}`),
        	pattern2 : regexp.MustCompile(config.Target_match),
		next:   next,
		name:   name,
	}, nil
}

func (a *SiteProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	destTemplate := a.pattern1.ReplaceAllString(a.config.Target_replace,url.QueryEscape(req.Header.Get(a.config.Header)))
	destination := a.pattern2.ReplaceAllString(req.URL, destTemplate)
	destinationUrl, err := url.Parse(destination)

	if err != nil {
		a.next.ServeHTTP(rw, req)
		return
	}

	proxy, found := a.cache.Get(destinationUrl)
        if !found {
		proxy = httputil.NewSingleHostReverseProxy(destinationUrl)
                a.cache.Add(destinationUrl,proxy)
	}
	proxy.ServeHTTP(rw, req)

	a.next.ServeHTTP(rw, req)
}
