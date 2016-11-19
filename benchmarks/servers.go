package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"testing"

	tchannel "github.com/uber/tchannel-go"
	traw "github.com/uber/tchannel-go/raw"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	yhttp "go.uber.org/yarpc/transport/http"
	ytch "go.uber.org/yarpc/transport/tchannel"
)

type serverConfig struct {
	impl         string
	transport    string
	encoding     string
	payloadBytes uint64
}

type benchServer interface {
	Start() (endpoint string, err error)
	Stop() error
}

type externalServer interface {
	benchServer
}

type localServer interface {
	benchServer
}

func yarpcEcho(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	return body, yarpc.NewResMeta().Headers(reqMeta.Headers()), nil
}

func httpEcho(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	hs := w.Header()
	for k, vs := range r.Header {
		hs[k] = vs
	}

	_, err := io.Copy(w, r.Body)
	if err != nil {
		panic(err)
	}
}

type tchannelEcho struct{ t testing.TB }

func (tchannelEcho) Handle(ctx context.Context, args *traw.Args) (*traw.Res, error) {
	return &traw.Res{Arg2: args.Arg2, Arg3: args.Arg3}, nil
}

func (t tchannelEcho) OnError(ctx context.Context, err error) {
	t.t.Fatalf("request failed: %v", err)
}

func newLocalServer(cfg serverConfig) localServer {
	if cfg.impl == "yarpc" {
		switch cfg.transport {
		case "tchannel":
			return newLocalYarpcTChannelServer(cfg)
		case "http":
			return newLocalYarpcHTTPServer(cfg)
		}
		log.Panicf("unknown transport %s", cfg.transport)
	}
	switch cfg.transport {
	case "tchannel":
		return newLocalTChannelServer(cfg)
	case "http":
		return newLocalHTTPServer(cfg)
	}
	log.Panicf("unknown transport %s", cfg.transport)
	return nil
}

type yarpcHTTPServer struct {
	inbound yhttp.Inbound
	disp    yarpc.Dispatcher
}

func newLocalYarpcHTTPServer(cfg serverConfig) localServer {
	httpInbound := yhttp.NewInbound("localhost:0")
	yarpcConfig := yarpc.Config{
		Name:     "bench_server",
		Inbounds: []transport.Inbound{httpInbound},
	}
	disp := yarpc.NewDispatcher(yarpcConfig)
	disp.Register(raw.Procedure("echo", yarpcEcho))
	return &yarpcHTTPServer{
		inbound: httpInbound,
		disp:    disp,
	}
}

func (s *yarpcHTTPServer) Start() (string, error) {
	if err := s.disp.Start(); err != nil {
		return "", err
	}
	return s.inbound.Addr().String(), nil
}

func (s *yarpcHTTPServer) Stop() error {
	return s.disp.Stop()
}

type yarpcTChannelServer struct {
	tch  *tchannel.Channel
	disp yarpc.Dispatcher
}

func newLocalYarpcTChannelServer(cfg serverConfig) localServer {
	tch, err := tchannel.NewChannel("bench_server", nil)
	if err != nil {
		panic(err)
	}
	tchInboud := ytch.NewInbound(tch, ytch.ListenAddr("localhost:0"))
	yarpcConfig := yarpc.Config{
		Name:     "bench_server",
		Inbounds: []transport.Inbound{tchInboud},
	}
	disp := yarpc.NewDispatcher(yarpcConfig)
	disp.Register(raw.Procedure("echo", yarpcEcho))
	return &yarpcTChannelServer{
		tch:  tch,
		disp: disp,
	}
}

func (s *yarpcTChannelServer) Start() (string, error) {
	if err := s.disp.Start(); err != nil {
		return "", err
	}
	return s.tch.PeerInfo().HostPort, nil
}

func (s *yarpcTChannelServer) Stop() error {
	return s.disp.Stop()
}

func newLocalTChannelServer(cfg serverConfig) localServer {
	log.Panicf("not implemented")
	return nil
}

type httpServer struct {
	l    net.Listener
	done <-chan struct{}
}

func newLocalHTTPServer(cfg serverConfig) localServer {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	return &httpServer{
		l: l,
	}
}

func (s *httpServer) Start() (string, error) {
	done := make(chan struct{})
	s.done = done
	go func() {
		if err := http.Serve(s.l, http.HandlerFunc(httpEcho)); err != nil {
			panic(err)
		}
		close(done)
	}()
	return s.l.Addr().String(), nil
}

func (s *httpServer) Stop() error {
	err := s.l.Close()
	<-s.done
	return err
}
