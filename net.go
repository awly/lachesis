package main

import (
	"encoding/gob"
	"errors"
	"flag"
	"log"
	"net"
	"os/exec"
	"strings"
	"time"
)

const (
	msgPing = iota
	msgOK
	msgErr
	msgExec
)

var (
	listenPort   = flag.Int("l", 1111, "port to listen on for inter-node communications")
	joinAddr     = flag.String("j", "", "address of any existing node to join")
	pingInterval = flag.Int("i", 5, "inter-node ping interval, seconds")
	timeout      = flag.Int("t", 10, "communication timeout, seconds")
)

type node struct {
	*net.TCPAddr
	LastSeen    time.Time
	failedPings int
	monitoring  bool
	Up          bool
}

type message struct {
	remote     *net.TCPAddr
	Typ        int8
	Data       interface{}
	ListenPort int32
}

func netInit() {
	gob.Register(message{})
	gob.Register(node{})
	gob.Register(make(map[string]node))
	gob.Register(exec.Error{})
	gob.Register(errors.New(""))
}

func joinCluster(a string) {
	if a == "" {
		return
	}
	log.Println("joining", a)
	addr, err := net.ResolveTCPAddr("tcp", a)
	if err != nil {
		log.Println("failed to resolve", a, err)
		return
	}
	syncNode(addr)
}

func clusterInterface(a string) {
	l, err := net.Listen("tcp", a)
	if err != nil {
		log.Println(err)
		exit <- struct{}{}
		return
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			exit <- struct{}{}
			return
		}
		go handleMsg(c)
	}
}

func handleMsg(con net.Conn) {
	defer con.Close()
	con.SetDeadline(time.Now().Add(time.Second * time.Duration(*timeout)))
	dec := gob.NewDecoder(con)
	enc := gob.NewEncoder(con)
	inMsg := message{}
	outMsg := message{ListenPort: int32(*listenPort)}
	err := dec.Decode(&inMsg)
	if err != nil {
		log.Println(err)
		return
	}

	remoteNode := con.RemoteAddr().(*net.TCPAddr)
	remoteNode.Port = int(inMsg.ListenPort)
	config.Lock()
	if n, ok := config.nodes[remoteNode.String()]; ok && !n.Up {
		n.Up = true
		config.nodes[remoteNode.String()] = n
		log.Println(remoteNode, "is back online")
	}
	config.Unlock()
	go syncNode(remoteNode)

	switch inMsg.Typ {
	case msgPing:
		outMsg.Data = cpNodes()
		outMsg.Typ = msgOK
	case msgExec:
		cmdStr := strings.Fields(inMsg.Data.(string))
		var cmd *exec.Cmd
		if len(cmdStr) > 1 {
			cmd = exec.Command(cmdStr[0], cmdStr[1:]...)
		} else {
			cmd = exec.Command(cmdStr[0])
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			outMsg.Data = err.Error()
		} else {
			outMsg.Data = string(output)
		}
		outMsg.Typ = msgOK
	default:
		log.Println("received unknown message", inMsg)
		outMsg.Typ = msgErr
		outMsg.Data = "unknown message type"
	}
	err = enc.Encode(outMsg)
	if err != nil {
		log.Println(err)
	}
}

func sendMsg(m message) (*message, error) {
	m.ListenPort = int32(*listenPort)
	con, err := net.DialTCP("tcp", nil, m.remote)
	if err != nil {
		return nil, err
	}
	con.SetDeadline(time.Now().Add(time.Second * time.Duration(*timeout)))
	defer con.Close()
	enc := gob.NewEncoder(con)
	dec := gob.NewDecoder(con)
	err = enc.Encode(m)
	if err != nil {
		return nil, err
	}
	resp := &message{remote: m.remote}
	err = dec.Decode(resp)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func ping(addr *net.TCPAddr) {
	// check if not adding ourselves
	if addr.Port == *listenPort {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			log.Println(err)
			exit <- struct{}{}
			return
		}
		for _, laddr := range addrs {
			if lipaddr, ok := laddr.(*net.IPNet); ok {
				if lipaddr.IP.Equal(addr.IP) {
					return
				}
			}
		}
	}
	for {
		resp, err := sendMsg(message{remote: addr, Typ: msgPing})
		config.Lock()
		n := config.nodes[addr.String()]
		if err != nil {
			log.Println("failed to contact", addr)
			n.failedPings++
			if n.failedPings > 2 {
				n.Up = false
				n.monitoring = false
				config.nodes[addr.String()] = n
				config.Unlock()
				log.Println(addr, "is unreachable")
				return
			}
		} else {
			if resp.Typ != msgOK {
				log.Println(addr, "->", resp.Data)
			}
			if nodes, ok := resp.Data.(map[string]node); ok {
				for _, newn := range nodes {
					go syncNode(newn.TCPAddr)
				}
			}
			n.LastSeen = time.Now()
			n.failedPings = 0
			n.Up = true
		}
		config.nodes[addr.String()] = n
		config.Unlock()
		time.Sleep(time.Second * time.Duration(*pingInterval))
	}
}

func syncNode(addr *net.TCPAddr) {
	config.Lock()
	defer config.Unlock()
	n, ok := config.nodes[addr.String()]
	if !ok {
		log.Println("adding new node", addr)
		n = node{TCPAddr: addr, Up: true}
	}
	if !n.monitoring && n.Up {
		n.monitoring = true
		n.failedPings = 0
		go ping(addr)
	}
	config.nodes[addr.String()] = n
}
