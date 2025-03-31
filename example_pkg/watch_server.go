package example_pkg

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type WatchServer struct {
	server   *rpc.Server
	port     uint16
	listener net.Listener
	handler  http.Handler
	srvMux   *http.ServeMux
	context  map[string]interface{}
}

func NewWatchServer(port uint16) *WatchServer {
	s := new(WatchServer)
	s.port = port
	s.srvMux = http.NewServeMux()
	s.context = make(map[string]interface{})
	return s
}

func (s *WatchServer) Register(
	pattern string,
	channel interface{},
	object interface{},
	handlerRequest func(http.ResponseWriter, *http.Request),
	handlerChannel func(interface{}, interface{}),
) {
	if channel != nil && object != nil {
		s.context[pattern] = object
		go handlerChannel(channel, object)
	}
	s.srvMux.Handle(pattern, http.HandlerFunc(handlerRequest))
}

func (s *WatchServer) Serve() {
	addr := fmt.Sprintf(":%d", s.port)
	e := http.ListenAndServe(addr, s.srvMux)
	if e != nil {
		log.Println(e.Error())
	}
}

func (s *WatchServer) Close() {
	e := s.listener.Close()
	if e != nil {
		log.Println(e.Error())
	}
}
