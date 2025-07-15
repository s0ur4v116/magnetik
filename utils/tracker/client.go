package tracker

import (
	"fmt"
	"magnetik/utils/torrent"
	"math/rand/v2"
)

func (c *Client) DiscoverPeers(torrent *torrent.TorrentFile) ([]Peer, error) {
	// this is the first request
	// must be of event "started"
	req := &AnnounceParams{
		InfoHash:   torrent.InfoHash,
		PeerID:     c.PeerID,
		Port:       c.HTTPPort,
		Uploaded:   0,
		Downloaded: 0,
		Left:       torrent.TotalLength(),
		Compact:    true,
		Event:      "started",
	}

	resp, err := c.Announce(torrent.Announce, req)
	if err != nil {
		return nil, fmt.Errorf("failed to announce to tracker: %w", err)
	}

	// shuffle peers for better distribution
	rand.Shuffle(len(resp.Peers), func(i, j int) {
		resp.Peers[i], resp.Peers[j] = resp.Peers[j], resp.Peers[i]
	})

	return resp.Peers, nil
}

func (p *Peer) String() string {
	return fmt.Sprintf("%s:%d", p.IP.String(), p.Port)
}
