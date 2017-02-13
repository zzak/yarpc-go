package peer_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/peer/x/circus"
	"go.uber.org/yarpc/peer/x/peerheap"
	"go.uber.org/yarpc/peer/x/roundrobin"
	"go.uber.org/yarpc/transport/http"
)

func BenchmarkCircus(b *testing.B) {
	x := http.NewTransport()
	pl := circus.New(x)
	pl.Start()
	defer pl.Stop()
	benchPeerChooser(b, pl, pl)
}

func BenchmarkPeerHeap(b *testing.B) {
	x := http.NewTransport()
	pl := peerheap.New(x)
	pl.Start()
	defer pl.Stop()
	benchPeerChooser(b, pl, pl)
}

func BenchmarkRoundRobin(b *testing.B) {
	x := http.NewTransport()
	pl := roundrobin.New(x)
	pl.Start()
	defer pl.Stop()
	benchPeerChooser(b, pl, pl)
}

func benchPeerChooser(b *testing.B, pc peer.Chooser, pl peer.List) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	GenerateLoad(ctx, pc)
	GenerateChaos(ctx, pl)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, finish, _ := pc.Choose(ctx, nil)
		finish(nil)
	}
}

func GenerateLoad(ctx context.Context, pc peer.Chooser) {
	for n := 0; n < 100; n++ {
		go worker(ctx, pc, false)
	}
}

func GenerateLoadLoud(ctx context.Context, pc peer.Chooser) {
	for n := 0; n < 100; n++ {
		go worker(ctx, pc, true)
	}
}

func worker(ctx context.Context, pc peer.Chooser, loud bool) {
	for {
		if loud {
			fmt.Printf(".")
		}
		_, finish, err := pc.Choose(ctx, nil)
		if err != nil {
			return
		}
		time.Sleep(time.Nanosecond * time.Duration(rand.Int31n(10)))
		finish(nil)
	}
}

func GenerateChaos(ctx context.Context, pl peer.List, size int) {
	// Populate the peer chooser with peer identifiers.
	cluster := make([]peer.Identifier, 0, size)
	for n := 0; n < size; n++ {
		pid := hostport.PeerIdentifier(fmt.Sprintf("127.0.0.%d", n))
		cluster = append(cluster, pid)
	}
	pl.Update(peer.ListUpdates{Additions: cluster})
}
