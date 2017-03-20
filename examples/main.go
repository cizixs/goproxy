package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/cizixs/goproxy"
)

const (
	defaultPort = ":8000"
)

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
	config := &goproxy.ProxyConfig{
		Targets: backends,
		Debug:   *debug,
		Prefix:  "/networks/",
	}

	proxy, err := goproxy.NewProxy(config)
	if err != nil {
		fmt.Println("Can not setup proxy. Exit...")
		return
	}

	http.Handle("/", proxy)
	fmt.Printf("Start Serving at %s\n", *port)
	fmt.Printf("Packets will forward to %s\n\n", backends)
	http.ListenAndServe(*port, nil)
}
