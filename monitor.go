package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
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
		candidates := make(map[string]struct{})

		if cfg.AliveOnly {
			for _, ip := range ps.Static() {
				candidates[ip.String()] = struct{}{}
			}
		}

		if ps.mode == "dynamic" {
			connected, ok := connectedPeerIPs(cfg)
			if !ok {
				return
			}
			for _, ip := range connected {
				candidates[ip] = struct{}{}
			}
		}

		var ips []string
		for ip := range candidates {
			ips = append(ips, ip)
		}

		if cfg.AliveOnly {
			ips = reachablePeerIPs(ips, cfg.P2PPort, time.Duration(cfg.CheckTimeout)*time.Second)
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

func connectedPeerIPs(cfg *NodeConfig) ([]string, bool) {
	secretBytes, err := os.ReadFile(cfg.Secret)
	if err != nil {
		log.Printf("monitor: failed to read secret file: %v", err)
		return nil, false
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
		return nil, false
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("grin", secret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("monitor: failed to contact node: %v", err)
		return nil, false
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var pr peerResponse
	if err := json.Unmarshal(data, &pr); err != nil {
		log.Printf("monitor: failed to parse response: %v", err)
		return nil, false
	}

	var ips []string
	for _, p := range pr.Result.Ok {
		host, _, err := net.SplitHostPort(p.Addr)
		if err == nil && net.ParseIP(host).To4() != nil {
			ips = append(ips, host)
		}
	}
	return ips, true
}

func reachablePeerIPs(ips []string, port int, timeout time.Duration) []string {
	var reachable []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 32)

	for _, ip := range ips {
		ip := ip
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			if !canDialPeer(ip, port, timeout) {
				return
			}

			mu.Lock()
			reachable = append(reachable, ip)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return reachable
}

func canDialPeer(ip string, port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
