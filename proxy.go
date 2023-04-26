// Package header_pattern_proxy
package multiplexer_proxy

import (
	"context"
	"fmt"
        "log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"github.com/patrickmn/go-cache"
	"time"
)

// Config the plugin configuration.
type Config struct {
	Header             string      `json:"header,omitempty" yaml:"Header" mapstructure:"Header" default:"X-Forward-User"`
        TargetMatch        string      `json:"target_match,omitempty" yaml:"Target_match" mapstructure:"Target_match" default:"^(.*)$"`
        TargetReplace      string      `json:"target_replace,omitempty" yaml:"Target_replace" mapstructure:"Target_replace" default:"test.$1"`
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
	        log.Printf("Plugin multiplexer-proxy Header zero-length")
		return nil, fmt.Errorf("header cannot be empty")
	}

	if len(config.TargetMatch) == 0 {
	        log.Printf("Plugin multiplexer-proxy TargetMatch zero-length")
		return nil, fmt.Errorf("target_match cannot be empty")
	}

	if len(config.TargetReplace) == 0 {
	        log.Printf("Plugin multiplexer-proxy TargetReplace zero-length")
		return nil, fmt.Errorf("target_replace cannot be empty")
	}

        log.Printf("Plugin multiplexer-proxy %s initialized: %s", name, config.TargetReplace)
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

	if req.Header.Get("X-Multiplexer-Proxy") == "true" {
              a.next.ServeHTTP(rw, req)
	      return
        }
	destTemplate := a.pattern1.ReplaceAllString(a.config.TargetReplace,url.QueryEscape(req.Header.Get(a.config.Header)))
	originalDest := req.Header.Get("X-Forwarded-Proto") + "://" + req.Host + req.URL.String()
        log.Printf("Plugin multiplexer-proxy called: %s %s %s",originalDest, destTemplate, a.pattern2.String())
	destination := a.pattern2.ReplaceAllString(originalDest, destTemplate)
        log.Printf("Plugin multiplexer-proxy called: %s",destination)

	destinationUrl, err := url.Parse(destination)

	if err != nil {
		a.next.ServeHTTP(rw, req)
		return
	}
	proxy, found := a.proxyCache.Get(destinationUrl.String())
        if !found {
		proxy = httputil.NewSingleHostReverseProxy(destinationUrl)
                a.proxyCache.Add(destinationUrl.String(),proxy,cache.DefaultExpiration)
	}
	req.Header["X-Multiplexer-Proxy"] = []string{"true"}
	req.Host = ""
	proxy.(*httputil.ReverseProxy).ServeHTTP(rw, req)
}
