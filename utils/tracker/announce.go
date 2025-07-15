package tracker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"magnetik/utils/bencode"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

func (c *Client) Announce(trackerURL string, req *AnnounceParams) (*AnnounceResponse, error) {
	u, err := url.Parse(trackerURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tracker url", err)
	}

	params := url.Values{}

	params.Add("info_hash", string(req.InfoHash[:]))
	params.Add("peer_id", string(req.PeerID[:]))
	params.Add("port", strconv.Itoa(req.Port))
	params.Add("uploaded", strconv.FormatInt(req.Uploaded, 10))
	params.Add("downloaded", strconv.FormatInt(req.Downloaded, 10))
	params.Add("left", strconv.FormatInt(req.Left, 10))

	if req.Compact {
		params.Add("compact", "1")
	} else {
		params.Add("compact", "0")
	}

	if req.Event != "" {
		params.Add("event", req.Event)
	}

	u.RawQuery = params.Encode()

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("%w: tracker not reachable", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read tracker response", err)
	}

	return parseAnnounceParams(body)
}

func parseAnnounceParams(body []byte) (*AnnounceResponse, error) {
	decoded, err := bencode.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tracker response", err)
	}

	dict, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%w: invalid tracker response", err)
	}

	if failureReason, ok := dict["failure reason"]; ok {
		reason, ok := failureReason.(string)
		if !ok {
			return nil, fmt.Errorf("invalid failure reason format from tracker")
		}

		return nil, fmt.Errorf("%s: tracker error", reason)
	}

	resp := &AnnounceResponse{}

	// I am assuming the tracker gave a good response
	if intervalVal, ok := dict["interval"]; ok {
		interval, ok := intervalVal.(int)
		if !ok {
			return nil, fmt.Errorf("invalid interval value")
		}

		resp.Interval = interval
	}

	if completeVal, ok := dict["complete"]; ok {
		complete, ok := completeVal.(int)
		if !ok {
			return nil, fmt.Errorf("invalid complete value")
		}

		resp.Complete = complete
	}

	if peersVal, ok := dict["peers"]; ok {
		switch peers := peersVal.(type) {
		case string:
			resp.Peers, err = parseCompactPeers([]byte(peers))
			if err != nil {
				return nil, fmt.Errorf("%w: failed to parse compact peers", err)
			}
		case []any:
			resp.Peers, err = parseNonCompactPeers(peers)
			if err != nil {
				return nil, fmt.Errorf("%w: failed to parse non-compact peers", err)
			}
		default:
			return nil, fmt.Errorf("invalid peers format")
		}
	}

	return resp, nil
}

func parseCompactPeers(body []byte) ([]Peer, error) {
	if len(body)%6 != 0 {
		return nil, fmt.Errorf("%d: invalid compact peers length", len(body))
	}

	numPeers := len(body) / 6
	peers := make([]Peer, numPeers)

	for i := range numPeers {
		offset := i * 6
		ip := net.IP(body[offset : offset+4])
		port := binary.BigEndian.Uint16(body[offset+4 : offset+6])

		peers[i] = Peer{
			IP:   ip,
			Port: int(port),
		}
	}

	return peers, nil
}

func parseNonCompactPeers(body []any) ([]Peer, error) {
	peers := make([]Peer, len(body))
	for i, peerData := range body {
		peerDict, ok := peerData.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("peer %d is not a dictionary", i)
		}

		if peerIDVal, ok := peerDict["peer id"]; ok {
			peerIDStr, ok := peerIDVal.(string)
			if !ok {
				return nil, fmt.Errorf("peer %d has invalid peer id", i)
			}

			copy(peers[i].ID[:], []byte(peerIDStr))
		}

		ipVal, ok := peerDict["ip"]
		if !ok {
			return nil, fmt.Errorf("peer %d has invalid ip", i)
		}

		ipStr, ok := ipVal.(string)
		if !ok {
			return nil, fmt.Errorf("peer %d has invalid ip", i)
		}

		peers[i].IP = net.ParseIP(ipStr)
		if peers[i].IP == nil {
			return nil, fmt.Errorf("peer %d has invalid ip address: %s", i, ipStr)
		}

		portVal, ok := peerDict["port"]
		if !ok {
			return nil, fmt.Errorf("peer %d missing port", i)
		}

		port, ok := portVal.(int64)
		if !ok {
			return nil, fmt.Errorf("peer %d has invalid port", i)
		}

		peers[i].Port = int(port)
	}

	return peers, nil
}
