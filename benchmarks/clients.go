package main

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
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

	client  raw.Client
	reqBody []byte
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
	client := raw.New(disp.Channel("bench_server"))
	reqBody := make([]byte, cfg.payloadBytes)
	rand.Read(reqBody)
	return &yarpcHTTPClient{
		Dispatcher: disp,
		client:     client,
		reqBody:    reqBody,
	}
}

func runYARPCClient(b *testing.B, c raw.Client) {
}

func (c *yarpcHTTPClient) Warmup() {
	b := testing.B{N: 10}
	c.RunBenchmark(&b)
}

func (c *yarpcHTTPClient) RunBenchmark(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, _, err := c.client.Call(ctx, yarpc.NewReqMeta().Procedure("echo"), c.reqBody)
		if err != nil {
			panic(err)
		}
	}
}
