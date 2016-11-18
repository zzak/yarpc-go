package main

import (
	"context"
	"crypto/rand"
	"log"
	"testing"
	"time"

	tchannel "github.com/uber/tchannel-go"

	"go.uber.org/yarpc"
	"go.uber.org/yarpc/encoding/raw"
	"go.uber.org/yarpc/transport"
	yhttp "go.uber.org/yarpc/transport/http"
	ytch "go.uber.org/yarpc/transport/tchannel"
)

type clientConfig struct {
	impl         string
	transport    string
	encoding     string
	payloadBytes uint64
	endpoint     string
}

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

type yarpcClientDispatcher struct {
	yarpc.Dispatcher

	reqBody []byte
}

func newLocalClient(cfg clientConfig) localClient {
	var outbound transport.UnaryOutbound

	switch cfg.transport {
	case "http":
		outbound = yhttp.NewOutbound("http://" + cfg.endpoint)
	case "tchannel":
		tch, err := tchannel.NewChannel("bench_server", nil)
		if err != nil {
			panic(err)
		}
		outbound = ytch.NewOutbound(tch, ytch.HostPort(cfg.endpoint))
	default:
		log.Panicf("unknown transport %s", cfg.transport)
		return nil
	}

	yarpcConfig := yarpc.Config{
		Name: "bench_client",
		Outbounds: yarpc.Outbounds{
			"bench_server": {
				Unary: outbound,
			},
		},
	}
	disp := yarpc.NewDispatcher(yarpcConfig)
	reqBody := make([]byte, cfg.payloadBytes)
	rand.Read(reqBody)

	clientDisp := yarpcClientDispatcher{
		Dispatcher: disp,
		reqBody:    reqBody,
	}

	switch cfg.encoding {
	case "raw":
		client := raw.New(disp.Channel("bench_server"))
		return &yarpcRawClient{
			yarpcClientDispatcher: clientDisp,
			client:                client,
		}
	default:
		log.Panicf("unknown encoding %s", cfg.encoding)
		return nil
	}
}

type yarpcRawClient struct {
	yarpcClientDispatcher

	client raw.Client
}

func (c *yarpcRawClient) Warmup() {
	b := testing.B{N: 10}
	c.RunBenchmark(&b)
}

func (c *yarpcRawClient) RunBenchmark(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		_, _, err := c.client.Call(ctx, yarpc.NewReqMeta().Procedure("echo"), c.reqBody)
		if err != nil {
			panic(err)
		}
	}
}
