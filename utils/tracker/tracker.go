package tracker

import "net"

type Client struct {
	PeerID   [20]byte
	HTTPPort int
}

func NewClient(peerID [20]byte, port int) *Client {
	return &Client{
		PeerID:   peerID,
		HTTPPort: port,
	}
}

type Peer struct {
	ID   [20]byte
	IP   net.IP
	Port int
}

type AnnounceParams struct {
	InfoHash   [20]byte
	PeerID     [20]byte
	Port       int
	Uploaded   int64
	Downloaded int64
	Left       int64
	Compact    bool
	Event      string
}

type AnnounceResponse struct {
	Interval   int
	Peers      []Peer
	Complete   int
	Incomplete int
}
