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

var valid = struct {
	transports stringSet
	clients    stringSet
	servers    stringSet
	encodings  stringSet
	payloads   stringSet
}{
	transports: map[string]struct{}{
		"tchannel": {},
		"http":     {},
	},
	clients: map[string]struct{}{
		"yarpc":  {},
		"native": {},
	},
	servers: map[string]struct{}{
		"yarpc":  {},
		"native": {},
	},
	encodings: map[string]struct{}{
		"raw":    {},
		"json":   {},
		"thrift": {},
	},
	payloads: map[string]struct{}{
		"16b":   {},
		"64b":   {},
		"512b":  {},
		"1kib":  {},
		"4kib":  {},
		"64kib": {},
	},
}

type flagSet struct {
	values []string
	valid  *stringSet
}

func newFlagSet(valid *stringSet) flagSet {
	return flagSet{valid: valid}
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
		if _, ok := (*fs.valid)[v]; !ok {
			return fmt.Errorf("Invalid value %q (choose from: %s)", v,
				strings.Join(fs.valid.Keys(), ", "))
		}
		fs.values = append(fs.values, v)
	}
	return nil
}

func flagSetVar(fs *flagSet, name string) {
	flag.Var(fs, name,
		fmt.Sprintf("comma separated list of %s to use (default: %s)", name,
			strings.Join(fs.valid.Keys(), ", ")))
}

func main() {
	var (
		spawn = flag.String("spawn", "",
			"internally used to spawn benchmark client/server on different processes")

		transports = newFlagSet(&valid.transports)
		clients    = newFlagSet(&valid.clients)
		servers    = newFlagSet(&valid.servers)
		encodings  = newFlagSet(&valid.encodings)
		payloads   = newFlagSet(&valid.payloads)
	)
	flagSetVar(&transports, "transports")
	flagSetVar(&clients, "clients")
	flagSetVar(&servers, "servers")
	flagSetVar(&encodings, "encodings")
	flagSetVar(&payloads, "payloads")

	flag.Parse()

	if *spawn == "" {
		benchMain()
	} else {
		spawnPeer()
	}
}

func benchMain() {
	log.Printf("starting benchmarks")
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
