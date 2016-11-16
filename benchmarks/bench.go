package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"testing"
)

type stringSet map[string]struct{}

func (ss *stringSet) Keys() []string {
	keys := make([]string, 0, len(*ss))
	for k := range *ss {
		keys = append(keys, k)
	}
	return keys
}

type flagSet struct {
	values []string
	valids stringSet
}

func (fs *flagSet) String() string {
	return strings.Join(fs.values, ", ")
}

func (fs *flagSet) Set(value string) error {
	var values = make(map[string]struct{})
	for _, v := range strings.Split(value, ",") {
		v := strings.Trim(v, " ")
		values[v] = struct{}{}
	}
	if len(fs.values) > 0 {
		return fmt.Errorf("Duplicate use of flag")
	}
	if len(values) == 0 {
		return fmt.Errorf("At least one value must be specified")
	}
	for v := range values {
		if _, ok := fs.valids[v]; !ok {
			return fmt.Errorf("Invalid value %q (choose from: %s)", v,
				strings.Join(fs.valids.Keys(), ", "))
		}
		fs.values = append(fs.values, v)
	}
	return nil
}

func flagStringSet(name string, valids []string) *flagSet {
	fs := flagSet{valids: stringSet{}}
	for _, v := range valids {
		fs.valids[v] = struct{}{}
	}
	flag.Var(&fs, name,
		fmt.Sprintf("comma separated list of %s to use (default: %s)", name,
			strings.Join(fs.valids.Keys(), ", ")))
	return &fs
}

var (
	flagSpawn = flag.String("spawn", "", "spawn a client/server")

	flagTransports = flagStringSet("transports", []string{
		"tchannel", "http",
	})
	flagClients = flagStringSet("clients", []string{
		"yarpc", "direct",
	})
	flagServers = flagStringSet("servers", []string{
		"yarpc", "direct",
	})
	flagEncodings = flagStringSet("encodings", []string{
		"raw", "json", "thrift",
	})
	flagPayloads = flagStringSet("payloads", []string{
		"16b", "64b", "512b", "1kib", "4kib", "64kib",
	})

	flagSpawnClient = flag.Bool("spawn_client", false,
		"spawn external process for client instead of server (useful for profiling the server during a benchmark)")
)

func main() {
	flag.Parse()

	if *flagSpawn == "" {
		benchMain()
	} else {
		spawnPeer()
	}
}

func benchMain() {
	log.Printf("Running benchmark for:")
	bcfg := benchConfig{
		impl:        "yarpc",
		transport:   "http",
		encoding:    "raw",
		payloadSize: "16b",
	}

	server := newLocalServer(bcfg)
	endpoint, err := server.Start()
	if err != nil {
		panic(err)
	}

	client := newLocalClient(bcfg, endpoint)
	err = client.Start()
	if err != nil {
		panic(err)
	}

	client.Warmup()
	result := testing.Benchmark(client.RunBenchmark)

	fmt.Printf("-> %s", result)
}

func spawnPeer() {
}
