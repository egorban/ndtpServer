package main

import (
	"flag"

	"github.com/egorban/ndtpServer/pkg/ndtpserv"
)

func main() {
	var listenAddress string
	flag.StringVar(&listenAddress, "l", "localhost:9001", "listen address (e.g. 'localhost:9001')")
	flag.Parse()
	ndtpserv.Start(listenAddress)
}
