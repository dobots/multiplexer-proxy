// Package header_pattern_proxy
package multiplexer_proxy

import (
	"context"
	"fmt"
        "log"
        "strings"
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
        headerPattern *regexp.Regexp
        targetPattern *regexp.Regexp
        dotPattern *regexp.Regexp
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
		headerPattern : regexp.MustCompile(`\${header}`),
        	targetPattern : regexp.MustCompile(config.TargetMatch),
                dotPattern : regexp.MustCompile(`\.`),
		next:   next,
		name:   name,
	}, nil
}

func (a *SiteProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	if req.Header.Get("X-Multiplexer-Proxy") == "true" {
	      log.Printf("Plugin multiplexer-proxy: skipping second round")
              a.next.ServeHTTP(rw, req)
	      return
        }
	destTemplate := a.headerPattern.ReplaceAllString(a.config.TargetReplace,a.dotPattern.ReplaceAllString(url.QueryEscape(strings.Replace(req.Header.Get(a.config.Header),"@","-at-",-1)),"-"))
	originalDest := req.Header.Get("X-Forwarded-Proto") + "://" + req.Host + req.URL.String()
        log.Printf("Plugin multiplexer-proxy called: %s %s %s",originalDest, destTemplate, a.targetPattern.String())
	destination := a.targetPattern.ReplaceAllString(originalDest, destTemplate)
        log.Printf("Plugin multiplexer-proxy: %s",destination)

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
