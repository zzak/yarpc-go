package peertest

import (
	"fmt"
	"math/rand"
	"time"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
	"golang.org/x/net/context"
)

func GenerateLoad(ctx context.Context, pc peer.Chooser, count int) {
	for n := 0; n < count; n++ {
		go worker(ctx, pc, false)
	}
}

func GenerateLoadLoud(ctx context.Context, pc peer.Chooser, count int) {
	for n := 0; n < count; n++ {
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
