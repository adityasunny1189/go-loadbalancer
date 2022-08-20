package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type simpleServer struct {
	addr string
	proxy *httputil.ReverseProxy
}

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type LoadBalancer struct {
	port string
	roundRobinCount int
	servers []Server 
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		roundRobinCount: 0,
		port: port,
		servers: servers,
	}
}

func newSimpleServer(addr string) *simpleServer {
	serverURL, err := url.Parse(addr)
	handleErr(err)

	return &simpleServer{
		addr: addr,
		proxy: httputil.NewSingleHostReverseProxy(serverURL),
	}
}

func handleErr(err error) {
	if err != nil {
		log.Panicf("error: %v\n", err)
		os.Exit(1)
	}
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	server := lb.servers[lb.roundRobinCount%len(lb.servers)]
	for !server.IsAlive() {
		lb.roundRobinCount++
		server = lb.servers[lb.roundRobinCount%len(lb.servers)]
	}
	lb.roundRobinCount++
	return server
} 

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNextAvailableServer()
	log.Printf("forwarding request to address %q\n", targetServer.Address())
	targetServer.Serve(rw, req)
}

func (s *simpleServer) Address() string {
	return s.addr
}

func (s *simpleServer) IsAlive() bool {
	return true
}

func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("https://www.bing.com"),
		newSimpleServer("https://www.duckduckgo.com"),
	}
	lb := NewLoadBalancer("8000", servers)
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}
	http.HandleFunc("/", handleRedirect)
	log.Printf("serving request at 'localhost:%s'\n", lb.port)
	http.ListenAndServe(":" + lb.port, nil)
}