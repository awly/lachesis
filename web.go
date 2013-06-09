package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var (
	httpPort = flag.Int("w", 8080, "address for web-interface")
	resDir   = flag.String("r", "res", "path to the web resources directory")
	tmpl     *template.Template
)

func webInit() {
	http.Handle("/res/", http.StripPrefix("/res/", http.FileServer(http.Dir(*resDir))))
	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/exec", handleExec)
	http.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		http.ServeFile(rw, req, *resDir+string(os.PathSeparator)+"home.html")
	})
}

// web interface functions
func webInterface(a string) {
	var err error
	tmpl, err = template.ParseFiles(*resDir + string(os.PathSeparator) + "table.html")
	if err != nil {
		log.Fatalln(err)
	}
	if err := http.ListenAndServe(a, nil); err != nil {
		log.Println(err)
		exit <- struct{}{}
	}
}

func handleStatus(rw http.ResponseWriter, req *http.Request) {
	config.RLock()
	nodes := make([]node, 0, len(config.nodes))
	for _, n := range config.nodes {
		nodes = append(nodes, n)
	}
	config.RUnlock()
	err := tmpl.Execute(rw, nodes)
	if err != nil {
		log.Println(err)
	}
}

func handleExec(rw http.ResponseWriter, req *http.Request) {
	command := req.FormValue("input")
	if command == "" {
		fmt.Fprint(rw, "Nothing to execute<br/>")
		return
	}
	regexStr := req.FormValue("regexp")
	if regexStr == "" {
		regexStr = ".*"
	}
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		fmt.Fprint(rw, err, "<br/>")
		return
	}
	nodes := cpNodes()
	reqSent := 0
	res := make(chan *message)
	for _, n := range nodes {
		if n.Up && regex.MatchString(n.String()) {
			reqSent++
			go func(to *net.TCPAddr) {
				resp, err := sendMsg(message{remote: to, Typ: msgExec, Data: command})
				if err != nil {
					resp = &message{remote: to, Typ: msgErr, Data: err.Error()}
				}
				res <- resp
			}(n.TCPAddr)
		}
	}
	if reqSent == 0 {
		fmt.Fprint(rw, "No nodes match given regular expression<br/>")
		return
	}
	for i := 0; i < reqSent; i++ {
		resp := <-res
		if respStr, ok := resp.Data.(string); ok {
			resp.Data = strings.Replace(respStr, "\n", "<br/>", -1)
		}
		_, err := fmt.Fprint(rw, "node:", resp.remote, "<br/>response:", resp.Data, "<hr/>")
		if err != nil {
			log.Println(err)
			return
		}
	}
}
