package server

import (
	"context"
	"log"
	"net/http"
)

type HTTPServer struct {
	srvMux *http.ServeMux
	server *http.Server

	listenAddr string

	// 用于接入外部的Server
	HandlerMap map[string]http.HandlerFunc
}

func NewHTTPServer(listenAddr string) *HTTPServer {
	return &HTTPServer{
		srvMux:     http.NewServeMux(),
		listenAddr: listenAddr,
		HandlerMap: map[string]http.HandlerFunc{},
	}
}

func (s *HTTPServer) SetListenAddr(listenAddr string) {
	s.listenAddr = listenAddr
}

func (s *HTTPServer) RegisterHandler(path string, handler http.HandlerFunc) {
	log.Printf("register handler on addr[%s] for [%s]", s.listenAddr, path)
	s.srvMux.Handle(path, handler)
	s.HandlerMap[path] = handler
}

func (s *HTTPServer) StartHttpServer() error {
	if len(s.listenAddr) == 0 {
		log.Printf("listenAddr is empty, skip http server start")
		return nil
	} else if len(s.HandlerMap) == 0 {
		log.Printf("HandlerMap is empty, skip http server start")
		return nil
	}
	s.server = &http.Server{Handler: s.srvMux}
	log.Printf("start a http server for metadata transform and query at :%s", s.listenAddr)

	go func() {
		err := http.ListenAndServe(s.listenAddr, s.srvMux)
		if err != nil && err != http.ErrServerClosed {
			log.Printf("fetcher server stop with error: %v", err)
		}
	}()
	return nil
}

func (s *HTTPServer) Stop() error {
	if s.server != nil {
		return s.server.Shutdown(context.Background())
	}
	return nil
}
