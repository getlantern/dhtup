package dhtup

import (
	"context"
	"fmt"
	"io"

	"github.com/anacrolix/dht/v2/exts/getput"
	"github.com/anacrolix/dht/v2/krpc"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
)

type Resource struct {
	DhtTarget    krpc.ID
	Context      *Context
	FilePath     string
	WebSeedUrls  []string
	Salt         []byte
	MetainfoUrls []string
}

func (me Resource) Open(ctx context.Context) (
	ret io.ReadCloser,
	// The error is temporary, try again in a bit.
	temporary bool,
	err error,
) {
	temporary = true
	res, _, err := getput.Get(ctx, me.DhtTarget, me.Context.DhtServer, nil, me.Salt)
	if err != nil {
		err = fmt.Errorf("getting latest infohash: %w", err)
		return
	}
	var bep46Payload krpc.Bep46Payload
	err = bencode.Unmarshal(res.V, &bep46Payload)
	if err != nil {
		err = fmt.Errorf("unmarshalling bep46 payload: %w", err)
		return
	}
	// We might want to drop old torrents that we're not using anymore. Other config file names or
	// resources may hold references to shared torrents. For now, we can let the old torrents
	// accumulate because there shouldn't be much churn, and we can continue to seed them for other
	// peers.
	t, _ := me.Context.TorrentClient.AddTorrentOpt(torrent.AddTorrentOpts{
		InfoHash: bep46Payload.Ih,
	})
	// Add a backup method to obtain the torrent info.
	t.UseSources(me.MetainfoUrls)
	// Add a local seed for testing, assuming that announcing will fail to return our own IP.
	t.AddPeers([]torrent.PeerInfo{{
		Addr:    localhostPeerAddr{},
		Trusted: true,
	}})
	// An alternate source for the torrent data, since the first peer has no other peers to
	// bootstrap from.
	t.AddWebSeeds(me.WebSeedUrls)
	select {
	case <-t.GotInfo():
	case <-ctx.Done():
		err = fmt.Errorf("waiting for torrent info: %w", ctx.Err())
		return
	}
	var f *torrent.File
	for _, f = range t.Files() {
		// I think the opts fileName is just a base name, our torrent should be structured so that
		// the files sit in the root folder to match.
		if f.DisplayPath() == me.FilePath {
			break
		}
	}
	if f == nil {
		// Well this is awkward.
		err = fmt.Errorf("file not found in torrent")
		// Fixing this would require a republish, which would be on the typical publishing schedule.
		temporary = false
		return
	}
	ret = f.NewReader()
	// Everything good, use the default!
	temporary = false
	return
}

type localhostPeerAddr struct{}

func (localhostPeerAddr) String() string {
	return "localhost:42069"
}
