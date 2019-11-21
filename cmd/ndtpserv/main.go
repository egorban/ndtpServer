package main

import (
	"flag"
	"github.com/egorban/ndtpServer/pkg/ndtpserv"
)

func main() {
	var listenPort string
	var mode int
	var numPackets int
	flag.StringVar(&listenPort, "p", "9001", "listen port (e.g. '9001')")
	flag.IntVar(&mode, "m", 0, "mode (1 - master (default 0)")
	flag.IntVar(&numPackets, "n", 100, "num received packets (e.g. 100)")
	flag.Parse()
	ndtpserv.Start(listenPort, mode, numPackets)
}
