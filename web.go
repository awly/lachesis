package main

import (
	"flag"
	"log"
	"net/http"
)

var (
	httpPort = flag.Int("w", 8080, "address for web-interface")
	resDir   = flag.String("r", "res", "path to the web resources directory")
)

func webInit() {
	http.Handle("/res", http.FileServer(http.Dir(*resDir)))
	http.HandleFunc("/", handleWebHome)
	// other http handlers
}

// web interface functions
func webInterface(a string) {
	if err := http.ListenAndServe(a, nil); err != nil {
		log.Println(err)
		exit <- struct{}{}
	}
}

func handleWebHome(rw http.ResponseWriter, req *http.Request) {
	// serve homepage
}
