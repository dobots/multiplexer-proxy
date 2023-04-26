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
		TargetReplace: "${header}.$1",
	}
}

type SiteProxy struct {
	config *Config
        proxyCache  *cache.Cache
        pattern1 *regexp.Regexp
        pattern2 *regexp.Regexp
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
                proxyCache: cache.New(5*time.Minute, 10*time.Minute),
		pattern1 : regexp.MustCompile(`\${header}`),
        	pattern2 : regexp.MustCompile(config.TargetMatch),
		next:   next,
		name:   name,
	}, nil
}

func (a *SiteProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

        fmt.Printf("Plugin multiplexer-proxy called")
	destTemplate := a.pattern1.ReplaceAllString(a.config.TargetReplace,url.QueryEscape(req.Header.Get(a.config.Header)))
	destination := a.pattern2.ReplaceAllString(req.URL.String(), destTemplate)
	destinationUrl, err := url.Parse(destination)

        fmt.Printf("multiplexer-proxy: %s -> %s = %s",destTemplate,destination,destinationUrl.String())
	if err != nil {
		a.next.ServeHTTP(rw, req)
		return
	}

	proxy, found := a.proxyCache.Get(destinationUrl.String())
        if !found {
		proxy = httputil.NewSingleHostReverseProxy(destinationUrl)
                a.proxyCache.Add(destinationUrl.String(),proxy,cache.DefaultExpiration)
	}
	proxy.(*httputil.ReverseProxy).ServeHTTP(rw, req)

	a.next.ServeHTTP(rw, req)
}
