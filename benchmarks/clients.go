package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"io/ioutil"
	"log"
	"net"
	"net/http"
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
	if cfg.impl == "yarpc" {
		return newLocalYarpcClient(cfg)
	}

	switch cfg.transport {
	case "http":
		return newLocalHTTPClient(cfg)
	case "tchannel":
		return newLocalTChannelClient(cfg)
	default:
		log.Panicf("unknown transport %s", cfg.transport)
		return nil
	}
}

func newLocalYarpcClient(cfg clientConfig) localClient {
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

type httpRawClient struct {
	transport *http.Transport
	client    http.Client
	url       string
	reqBody   []byte
}

func newLocalHTTPClient(cfg clientConfig) localClient {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	reqBody := make([]byte, cfg.payloadBytes)
	rand.Read(reqBody)

	return &httpRawClient{
		transport: transport,
		client: http.Client{
			Transport: transport,
		},
		url:     "http://" + cfg.endpoint,
		reqBody: reqBody,
	}
}

func (c *httpRawClient) Start() error {
	return nil
}

func (c *httpRawClient) Stop() error {
	c.transport.CloseIdleConnections()
	return nil
}

func (c *httpRawClient) Warmup() {
	b := testing.B{N: 10}
	c.RunBenchmark(&b)
}

func (c *httpRawClient) RunBenchmark(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		req, err := http.NewRequest("POST", c.url, bytes.NewReader(c.reqBody))
		if err != nil {
			panic(err)
		}
		req = req.WithContext(ctx)
		req.Header = http.Header{
			"Context-TTL-MS": {"100"},
			"Rpc-Caller":     {"bench_client"},
			"Rpc-Encoding":   {"raw"},
			"Rpc-Procedure":  {"echo"},
			"Rpc-Service":    {"bench_server"},
		}
		res, err := c.client.Do(req)
		if err != nil {
			panic(err)
		}
		if _, err := ioutil.ReadAll(res.Body); err != nil {
			panic(err)
		}
		if err := res.Body.Close(); err != nil {
			panic(err)
		}
	}
}

func newLocalTChannelClient(cfg clientConfig) localClient {
	log.Panicf("not implemented")
	return nil
}
