package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"text/tabwriter"

	"code.cloudfoundry.org/bytefmt"
)

var allAxes = map[string][]string{
	"transports": {"tchannel", "http"},
	"clients":    {"yarpc", "direct"},
	"servers":    {"yarpc", "direct"},
	"encodings":  {"raw", "json", "thrift"},
	"payloads":   {"16b", "64b", "512b", "1kb", "4kb", "64kb"},
}

type stringSet map[string]struct{}

func (ss *stringSet) Keys() []string {
	keys := make([]string, 0, len(*ss))
	for k := range *ss {
		keys = append(keys, k)
	}
	return keys
}

func (ss stringSet) String() string {
	return strings.Join(ss.Keys(), ", ")
}

type flagSet struct {
	values []string
	Valids stringSet
}

func (fs *flagSet) Values() []string {
	if len(fs.values) == 0 {
		return fs.Valids.Keys()
	}
	return fs.values
}

func (fs flagSet) String() string {
	return strings.Join(fs.Values(), ", ")
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
		if _, ok := fs.Valids[v]; !ok {
			return fmt.Errorf("Invalid value %q (choose from: %s)",
				v, fs.Valids)
		}
		fs.values = append(fs.values, v)
	}
	return nil
}

func flagStringSet(name string) *flagSet {
	fs := flagSet{Valids: stringSet{}}
	for _, v := range allAxes[name] {
		fs.Valids[v] = struct{}{}
	}
	flag.Var(&fs, name,
		fmt.Sprintf("comma separated list of %s to use (default: %s)",
			name, fs.Valids))
	return &fs
}

var (
	flagSpawn = flag.String("spawn", "", "spawn a client/server")

	flagTransports = flagStringSet("transports")
	flagClients    = flagStringSet("clients")
	flagServers    = flagStringSet("servers")
	flagEncodings  = flagStringSet("encodings")
	flagPayloads   = flagStringSet("payloads")

	flagExtClient = flag.Bool("ext_client", false, "client as external process")
	flagExtServer = flag.Bool("ext_server", true, "server as external process")
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
	fmt.Println("Running benchmarks for:")
	axes := map[string][]string{
		"transport": flagTransports.Values(),
		"client":    flagClients.Values(),
		"server":    flagServers.Values(),
		"encoding":  flagEncodings.Values(),
		"payload":   flagPayloads.Values(),
	}
	combinations := Combinations(axes)

	{
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.AlignRight)
		fmt.Fprintln(w, "\t")
		fmt.Fprintln(w, "Transports\t:", flagTransports)
		fmt.Fprintln(w, "Clients\t:", flagClients)
		fmt.Fprintln(w, "Servers\t:", flagServers)
		fmt.Fprintln(w, "Encodingss\t:", flagEncodings)
		fmt.Fprintln(w, "Payloads\t:", flagPayloads)
		fmt.Fprintln(w, "\t")
		fmt.Fprintln(w, "Combinations\t:", len(combinations), "benchmark(s) to run")
		fmt.Fprint(w, "Processes\t: ")
		if *flagExtClient {
			fmt.Fprint(w, "out-of-process")
		} else {
			fmt.Fprint(w, "in-process")
		}
		fmt.Fprint(w, " client / ")
		if *flagExtServer {
			fmt.Fprint(w, "out-of-process")
		} else {
			fmt.Fprint(w, "in-process")
		}
		fmt.Fprint(w, " server")
		fmt.Fprintln(w)
		w.Flush()
	}

	var results []testing.BenchmarkResult
	for i, c := range combinations {
		msg := fmt.Sprintf("%d/%d (%d%%)", i+1, len(combinations),
			(i+1)*100/len(combinations))

		yarpcClient := (c["client"] == "yarpc")
		yarpcServer := (c["server"] == "yarpc")

		switch {
		case yarpcClient && yarpcServer:
			msg += fmt.Sprintf(" %s -> %s -> %s", c["client"], c["transport"], c["server"])
		case !yarpcClient && yarpcServer:
			msg += fmt.Sprintf(" %s -> %s", c["transport"], c["server"])
		case yarpcClient && !yarpcServer:
			msg += fmt.Sprintf(" %s -> %s", c["client"], c["transport"])
		case !yarpcClient && !yarpcServer:
			msg += fmt.Sprintf(" %s -> %s", c["transport"], c["transport"])
		}
		msg += fmt.Sprintf(" %s(%s)", c["encoding"], c["payload"])
		log.Print(msg)

		payloadBytes, err := bytefmt.ToBytes(c["payload"])
		if err != nil {
			panic(err)
		}
		cfg := benchConfig{
			client:       c["client"],
			server:       c["server"],
			transport:    c["transport"],
			encoding:     c["encoding"],
			payloadBytes: payloadBytes,
		}

		server := newLocalServer(cfg)
		endpoint, err := server.Start()
		if err != nil {
			panic(err)
		}

		client := newLocalClient(cfg, endpoint)
		err = client.Start()
		if err != nil {
			panic(err)
		}

		client.Warmup()
		result := testing.Benchmark(client.RunBenchmark)
		log.Printf("%s", result)
		results = append(results, result)
	}
}

func spawnPeer() {
}
