package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const (
	defaultPort = ":8000"
)

// GoProxy is our reverseproxy object
type GoProxy struct {
	target *url.URL
	proxy  *httputil.ReverseProxy
	debug  bool
}

// New creates a GoProxy instance
func New(backend string, debug bool) *GoProxy {
	url, err := url.Parse(backend)
	if err != nil {
		return nil
	}

	return &GoProxy{
		target: url,
		proxy:  httputil.NewSingleHostReverseProxy(url),
		debug:  debug,
	}
}

func (p *GoProxy) handle(w http.ResponseWriter, r *http.Request) {
	if p.debug {
		fmt.Println("--- New request ---")
		req, _ := httputil.DumpRequest(r, true)
		fmt.Println(string(req))
	}

	w.Header().Set("X-GoProxy", "GoProxy-by-cizixs")
	p.proxy.ServeHTTP(w, r)
}

func main() {
	port := flag.String("port", defaultPort, "Port for goproxy to run on.")
	backend := flag.String("backend", "", "Backend url address goproxy will forward packets to.")
	debug := flag.Bool("debug", true, "If enable debug mode. If so, application will print each request detail to stdout.")
	flag.Parse()

	if *backend == "" {
		fmt.Println("Must provide target url.\nUse --help to check usage.")
		return
	}

	proxy := New(*backend, *debug)
	if proxy == nil {
		fmt.Println("Can not setup proxy. Exit...")
		return
	}

	http.HandleFunc("/", proxy.handle)
	fmt.Printf("Start Serving at %s\n", *port)
	fmt.Printf("Packets will forward to %s\n\n", *backend)
	http.ListenAndServe(*port, nil)
}
