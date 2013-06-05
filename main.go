package main

import (
	"flag"
	"strconv"
	"sync"
)

var (
	exit   = make(chan struct{})
	config = struct {
		sync.RWMutex
		nodes map[string]node
	}{
		nodes: make(map[string]node),
	}
)

func init() {
	webInit()
	netInit()
}

func main() {
	flag.Parse()
	go webInterface(":" + strconv.Itoa(*httpPort))
	go clusterInterface(":" + strconv.Itoa(*listenPort))
	joinCluster(*joinAddr)
	<-exit
}
