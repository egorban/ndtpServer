package main

import (
	"flag"
	"github.com/egorban/ndtpServer/pkg/ndtpserv"
)

func main() {
	var listenPort string
	flag.StringVar(&listenPort, "p", "9001", "listen port (e.g. '9001')")
	flag.Parse()
	ndtpserv.Start(listenPort)
}
