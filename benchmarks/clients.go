package main

import (
	"testing"

	"go.uber.org/yarpc"
	yhttp "go.uber.org/yarpc/transport/http"
)

type benchClient interface {
	Start() error
	Stop() error

	Warmup()
	RunBenchmark(b *testing.B)
}

type externalClient interface {
	benchClient
}

type localClient interface {
	benchClient
}

type yarpcHTTPClient struct {
	yarpc.Dispatcher
}

func newLocalClient(cfg benchConfig, endpoint string) localClient {
	clientCfg := yarpc.Config{
		Name: "bench_client",
		Outbounds: yarpc.Outbounds{
			"bench_server": {
				Unary: yhttp.NewOutbound("http://" + endpoint),
			},
		},
	}
	disp := yarpc.NewDispatcher(clientCfg)
	return &yarpcHTTPClient{disp}
}

func (c yarpcHTTPClient) Warmup() {
}

func (c yarpcHTTPClient) RunBenchmark(b *testing.B) {
}
