package main

import (
	"context"
	"log"

	tchannel "github.com/uber/tchannel-go"

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
	tchInboud := ytch.NewInbound(tch)
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

func newLocalHTTPServer(cfg serverConfig) localServer {
	log.Panicf("not implemented")
	return nil
}
