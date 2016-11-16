package main

import (
	"context"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	yhttp "go.uber.org/yarpc/transport/http"
)

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

type yarpcHTTPServer struct {
	inbound yhttp.Inbound
	disp    yarpc.Dispatcher
}

func yarpcEcho(ctx context.Context, reqMeta yarpc.ReqMeta, body []byte) ([]byte, yarpc.ResMeta, error) {
	return body, yarpc.NewResMeta().Headers(reqMeta.Headers()), nil
}

func newLocalServer(cfg benchConfig) localServer {
	httpInbound := yhttp.NewInbound("localhost:0")
	serverCfg := yarpc.Config{
		Name:     "bench_server",
		Inbounds: []transport.Inbound{httpInbound},
	}
	disp := yarpc.NewDispatcher(serverCfg)
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
