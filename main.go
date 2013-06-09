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

func cpNodes() map[string]node {
	res := make(map[string]node)
	config.RLock()
	for k, v := range config.nodes {
		res[k] = v
	}
	config.RUnlock()
	return res
}

func main() {
	flag.Parse()
	webInit()
	netInit()
	go webInterface(":" + strconv.Itoa(*httpPort))
	go clusterInterface(":" + strconv.Itoa(*listenPort))
	joinCluster(*joinAddr)
	<-exit
}
