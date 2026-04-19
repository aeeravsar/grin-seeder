package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	ID      int    `json:"id"`
}

type peerInfo struct {
	Addr string `json:"addr"`
}

type peerResponse struct {
	Result struct {
		Ok []peerInfo `json:"Ok"`
	} `json:"result"`
}

func runMonitor(cfg *NodeConfig, ps *PeerState) {
	// static + !alive_only never needs the node
	if ps.mode == "static" && !ps.aliveOnly {
		return
	}

	poll := func() {
		secretBytes, err := os.ReadFile(cfg.Secret)
		if err != nil {
			log.Printf("monitor: failed to read secret file: %v", err)
			return
		}
		secret := strings.TrimSpace(string(secretBytes))

		body, _ := json.Marshal(rpcRequest{
			JSONRPC: "2.0",
			Method:  "get_connected_peers",
			Params:  []any{},
			ID:      1,
		})

		req, err := http.NewRequest("POST", cfg.URL+"/v2/owner", bytes.NewReader(body))
		if err != nil {
			log.Printf("monitor: failed to build request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.SetBasicAuth("grin", secret)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("monitor: failed to contact node: %v", err)
			return
		}
		defer resp.Body.Close()

		data, _ := io.ReadAll(resp.Body)

		var pr peerResponse
		if err := json.Unmarshal(data, &pr); err != nil {
			log.Printf("monitor: failed to parse response: %v", err)
			return
		}

		var ips []string
		for _, p := range pr.Result.Ok {
			host, _, err := net.SplitHostPort(p.Addr)
			if err == nil && net.ParseIP(host).To4() != nil {
				ips = append(ips, host)
			}
		}

		ps.UpdateAlive(ips)
		log.Printf("monitor: updated peer list — %d alive peers", len(ips))
	}

	poll()
	ticker := time.NewTicker(time.Duration(cfg.Interval) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		poll()
	}
}
