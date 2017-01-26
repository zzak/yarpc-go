package circus

import (
	"testing"

	"go.uber.org/yarpc/api/peer"
	"go.uber.org/yarpc/peer/hostport"
	"go.uber.org/yarpc/transport/http"
)

func Test(t *testing.T) {
	x := http.NewTransport()
	pl := New(x)
	pl.Update(peer.ListUpdates{
		Additions: []peer.Identifier{hostport.PeerIdentifier("127.0.0.1:80")},
	})
	pl.Start()
}
