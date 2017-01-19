package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const (
	defaultPort = ":8000"
)

// ProxyConfig contains all configurations for GoProxy
type ProxyConfig struct {
	// targets is a list of url, used as backends
	Targets []string

	// debug indicates if to enable debug mode.
	// If enabled, goproxy will log every request received
	Debug bool

	// Prefix is the path prefix to be removed when sending to backends
	// For example, if proxy is serving at http://example.com/,
	// Prefix is "/api/v1", and backend is "http://backend.com",
	// request to "http://example.com/api/v1/users/" will be transfered
	// to "http://backend.com/users/".
	Prefix string
}

// LoadBalancer is the load balance interface.
// The core function is to choose a backend to use from multiple backends
//
// TODO(wuwei): add more information to backends.
// Random or Hash lb algorithms are simple,
// but other lb algorithm might need backend load, connection info etc
// to make the right choice.
type LoadBalancer interface {
	choose([]*url.URL) *url.URL
}

// RandomLoadBalancer chooses a backend randomly
type RandomLoadBalancer struct{}

func (rb *RandomLoadBalancer) choose(targets []*url.URL) *url.URL {
	return targets[rand.Int()%len(targets)]
}

// GoProxy is our reverseproxy object
type GoProxy struct {
	targets []*url.URL
	proxy   *httputil.ReverseProxy
	debug   bool
	lb      *LoadBalancer
}

// NewProxy creates a GoProxy instance with specific configs
func NewProxy(config *ProxyConfig) (*GoProxy, error) {
	var targets []*url.URL
	for _, backend := range config.Targets {
		url, err := url.Parse(backend)
		if err != nil {
			return nil, err
		}
		targets = append(targets, url)
	}

	// TODO(wuwei): use loadbalancer in Config, then fallback to default
	lb := &RandomLoadBalancer{}
	director := func(req *http.Request) {
		target := lb.choose(targets)
		fmt.Printf("Target %s choosed\n", target.Host)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host

		// No need to check if request path starts with prefix, because
		// `strings.TrimPrefix` handles it as expected.
		if config.Prefix != "" {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, config.Prefix)
		}
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)

		// concatenate query strings of request and target
		if target.RawQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = target.RawQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = target.RawQuery + "&" + req.URL.RawQuery
		}
		fmt.Printf("Target path: %s\n", req.URL.Path)
	}

	return &GoProxy{
		targets: targets,
		proxy:   &httputil.ReverseProxy{Director: director},
		debug:   config.Debug,
	}, nil
}

func (p *GoProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.debug {
		fmt.Println("\n\n--- New request ---")
		req, _ := httputil.DumpRequest(r, true)
		fmt.Println(string(req))
	}

	w.Header().Set("X-GoProxy", "GoProxy-by-cizixs")
	p.proxy.ServeHTTP(w, r)
}

// singleJoiningSlash joins two string with proper slash.
// It is copied from `net/httputil/reverseproxy.go`
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func main() {
	port := flag.String("port", defaultPort, "Port for goproxy to run on.")
	backendStr := flag.String("backend", "", "Backend url address goproxy will forward packets to.")
	debug := flag.Bool("debug", true, "If enable debug mode. If so, application will print each request detail to stdout.")
	flag.Parse()

	if *backendStr == "" {
		fmt.Println("Must provide target url.\nUse --help to check usage.")
		return
	}

	backends := strings.Split(*backendStr, ",")
	config := &ProxyConfig{
		Targets: backends,
		Debug:   *debug,
		Prefix:  "/networks/",
	}

	proxy, err := NewProxy(config)
	if err != nil {
		fmt.Println("Can not setup proxy. Exit...")
		return
	}

	http.Handle("/networks/", proxy)
	fmt.Printf("Start Serving at %s\n", *port)
	fmt.Printf("Packets will forward to %s\n\n", backends)
	http.ListenAndServe(*port, nil)
}
