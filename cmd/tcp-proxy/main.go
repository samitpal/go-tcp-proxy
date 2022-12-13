//A tee'ing tcp proxy. The proxy takes two remote backends and sends all tcp traffic from clients to both the remote backends.
//The proxy sends the response from the first remote back to the client and discards the one received from the second remote
package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	proxy "github.com/jpillora/go-tcp-proxy"
)

var (
	version = "0.0.1"
	connid  = uint64(0)
	logger  proxy.ColorLogger

	localAddr        = flag.String("l", ":9999", "local address")
	remoteAddr       = flag.String("r1", "localhost:80", "first remote address. The client gets response back only from the first remote.")
	secondRemoteAddr = flag.String("r2", "localhost:8080", "second remote address. The proxy discards the response from this remote")
	verbose          = flag.Bool("v", false, "display server actions")
	veryverbose      = flag.Bool("vv", false, "display server actions and all tcp data")
	nagles           = flag.Bool("n", false, "disable nagles algorithm")
	hex              = flag.Bool("h", false, "output hex")
	colors           = flag.Bool("c", false, "output ansi colors")
)

func main() {
	flag.Parse()

	logger := proxy.ColorLogger{
		Verbose: *verbose,
		Color:   *colors,
	}

	logger.Info("go-tcp-proxy (%s) proxing from %v to %v and %v", version, *localAddr, *remoteAddr, *secondRemoteAddr)

	laddr, err := net.ResolveTCPAddr("tcp", *localAddr)
	if err != nil {
		logger.Warn("Failed to resolve local address: %s", err)
		os.Exit(1)
	}
	r1addr, err := net.ResolveTCPAddr("tcp", *remoteAddr)
	if err != nil {
		logger.Warn("Failed to resolve remote address: %s", err)
		os.Exit(1)
	}
	r2addr, err := net.ResolveTCPAddr("tcp", *secondRemoteAddr)
	if err != nil {
		logger.Warn("Failed to resolve second remote address: %s", err)
		os.Exit(1)
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		logger.Warn("Failed to open local port to listen: %s", err)
		os.Exit(1)
	}

	if *veryverbose {
		*verbose = true
	}

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			logger.Warn("Failed to accept connection '%s'", err)
			continue
		}
		connid++

		var p *proxy.Proxy
		p = proxy.New(conn, laddr, r1addr, r2addr)

		p.Nagles = *nagles
		p.OutputHex = *hex
		p.Log = proxy.ColorLogger{
			Verbose:     *verbose,
			VeryVerbose: *veryverbose,
			Prefix:      fmt.Sprintf("Connection #%03d ", connid),
			Color:       *colors,
		}

		go p.Start()
	}
}
