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

// GoProxy is our reverseproxy object
type GoProxy struct {
	targets []*url.URL
	proxy   *httputil.ReverseProxy
	debug   bool
}

// New creates a GoProxy instance
func New(backends []string, debug bool) *GoProxy {
	var targets []*url.URL
	for _, backend := range backends {
		url, err := url.Parse(backend)
		if err != nil {
			return nil
		}
		targets = append(targets, url)
	}

	director := func(req *http.Request) {
		target := targets[rand.Int()%len(targets)]
		fmt.Printf("Target %s choosed\n", target.Host)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
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
		debug:   debug,
	}
}

func (p *GoProxy) handle(w http.ResponseWriter, r *http.Request) {
	if p.debug {
		fmt.Println("\n\n--- New request ---")
		req, _ := httputil.DumpRequest(r, true)
		fmt.Println(string(req))
	}

	w.Header().Set("X-GoProxy", "GoProxy-by-cizixs")
	p.proxy.ServeHTTP(w, r)
}

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
	proxy := New(backends, *debug)
	if proxy == nil {
		fmt.Println("Can not setup proxy. Exit...")
		return
	}

	http.HandleFunc("/", proxy.handle)
	fmt.Printf("Start Serving at %s\n", *port)
	fmt.Printf("Packets will forward to %s\n\n", backends)
	http.ListenAndServe(*port, nil)
}
