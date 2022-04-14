package dhtup

import (
	"context"
	"fmt"
	"io"

	"github.com/anacrolix/dht/v2/exts/getput"
	"github.com/anacrolix/dht/v2/krpc"
	"github.com/anacrolix/missinggo"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

type OpenedResource interface {
	missinggo.ReadContexter
	io.Closer
}

type Resource interface {
	// Fetches the bep46 payload for this resource
	FetchBep46Payload(context.Context) (metainfo.Hash, error)
	// Makes a torrent out of the info in the Bep46Payload and returns the torrent's io.ReadCloser
	FetchTorrentFileReader(context.Context, metainfo.Hash) (OpenedResource, bool, error)
	// Fetches the bep46 payload for this resource, and returns the torrent's io.ReadCloser.
	// This is basically, running FetchBep46Payload() and then FetchTorrentFileReader(
	Open(ctx context.Context) (_ OpenedResource, temporary bool, _ error)
}

// ResourceImpl implements Resource
type ResourceImpl struct {
	ResourceInput
}

// ResourceInput is a typed constructor for Resource
type ResourceInput struct {
	DhtTarget    krpc.ID
	DhtContext   *Context
	FilePath     string
	WebSeedUrls  []string
	Salt         []byte
	MetainfoUrls []string
}

func NewResource(input ResourceInput) Resource {
	return &ResourceImpl{input}
}

func (me *ResourceImpl) FetchBep46Payload(ctx context.Context) (metainfo.Hash, error) {
	// TODO <22-03-22, soltzen> Have an option in this system to store the
	// current `seq` parameter and only download new ones
	res, _, err := getput.Get(ctx, me.ResourceInput.DhtTarget, me.ResourceInput.DhtContext.DhtServer, nil, me.ResourceInput.Salt)
	if err != nil {
		return metainfo.Hash{}, fmt.Errorf("getting latest infohash: %w", err)
	}
	bep46Payload := &krpc.Bep46Payload{}
	err = bencode.Unmarshal(res.V, bep46Payload)
	if err != nil {
		return metainfo.Hash{}, fmt.Errorf("unmarshalling bep46 payload: %w", err)
	}
	return bep46Payload.Ih, nil
}

func (me *ResourceImpl) FetchTorrentFileReader(ctx context.Context, bep46PayloadInfohash metainfo.Hash) (
	ret OpenedResource,
	// The error is temporary, try again in a bit.
	temporary bool,
	err error,
) {
	temporary = true
	// We might want to drop old torrents that we're not using anymore. Other config file names or
	// resources may hold references to shared torrents. For now, we can let the old torrents
	// accumulate because there shouldn't be much churn, and we can continue to seed them for other
	// peers.
	t, _ := me.ResourceInput.DhtContext.TorrentClient.AddTorrentOpt(torrent.AddTorrentOpts{
		InfoHash: bep46PayloadInfohash,
	})
	// Add a backup method to obtain the torrent info.
	t.UseSources(me.ResourceInput.MetainfoUrls)
	// Add a local seed for testing, assuming that announcing will fail to return our own IP.
	t.AddPeers([]torrent.PeerInfo{{
		Addr:    localhostPeerAddr{},
		Trusted: true,
	}})
	// An alternate source for the torrent data, since the first peer has no other peers to
	// bootstrap from.
	t.AddWebSeeds(me.ResourceInput.WebSeedUrls)
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
		if f.DisplayPath() == me.ResourceInput.FilePath {
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

func (me *ResourceImpl) Open(ctx context.Context) (
	ret OpenedResource,
	// The error is temporary, try again in a bit.
	temporary bool,
	err error,
) {
	temporary = true
	bep46Payload, err := me.FetchBep46Payload(ctx)
	if err != nil {
		err = fmt.Errorf("unmarshalling bep46 payload: %w", err)
		return
	}
	return me.FetchTorrentFileReader(ctx, bep46Payload)
}

type localhostPeerAddr struct{}

func (localhostPeerAddr) String() string {
	return "localhost:42069"
}
