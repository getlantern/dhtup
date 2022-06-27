package dhtup

import (
	"context"
	"encoding/hex"
	"testing"

	cryptoRand "crypto/rand"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/publicip"
	"github.com/anacrolix/torrent"
	"github.com/stretchr/testify/require"
)

// TestDhtGetPut tests PutBep46Payload() and GetBep46Payload() functions with
// the live BitTorrent DHT (no mocks).
func TestDhtGetPutWithName(t *testing.T) {
	ipv4, err := publicip.Get4(context.Background())
	require.NoError(t, err)
	dhtCfg := dht.NewDefaultServerConfig()
	dhtCfg.PublicIP = ipv4
	ds, err := dht.NewServer(nil)
	require.NoError(t, err)
	defer ds.Close()

	// Make some random keys and PUT them to the DHT
	name := "test"
	salt := make([]byte, 4)
	var privKey [32]byte
	var inputInfohash torrent.InfoHash
	_, err = cryptoRand.Read(salt)
	require.NoError(t, err)
	_, err = cryptoRand.Read(privKey[:])
	require.NoError(t, err)
	_, err = cryptoRand.Read(inputInfohash[:])
	require.NoError(t, err)

	dhtContext, err := NewContext(ipv4, t.TempDir())
	require.NoError(t, err)
	publisherRes := NewResource(ResourceInput{
		DhtContext:  &dhtContext,
		PrivKeySeed: privKey,
		Salt:        hex.EncodeToString(salt),
	})

	target, err := publisherRes.PutBep46PayloadWithName(
		context.Background(),
		"test",
		inputInfohash, // infohash
		0,             // seq
		true,          // autoseq
	)
	require.NoError(t, err)
	// fmt.Printf("target = %x\n", target)
	// fmt.Printf("salt = %x\n", salt)
	// fmt.Printf("inputInfohash = %x\n", inputInfohash)

	// Get the same target from the DHT and assert that the infohashes are the
	// same
	consumerRes := NewResource(ResourceInput{
		DhtContext: &dhtContext,
		DhtTarget:  target,
		Salt:       hex.EncodeToString(salt),
	})

	actualNameAndInfohash, err := consumerRes.FetchBep46PayloadWithName(
		context.Background(),
	)
	require.NoError(t, err)
	// fmt.Printf("actualInfohash = %+v\n", actualInfohash)
	require.NotEqual(t, actualNameAndInfohash, torrent.InfoHash{})
	require.Equal(t, inputInfohash[:], actualNameAndInfohash.Ih[:])
	require.Equal(t, name, actualNameAndInfohash.Name)
}
