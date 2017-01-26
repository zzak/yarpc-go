package peer

import (
	"context"
	"testing"
	"time"

	"go.uber.org/yarpc/api/peer"
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
