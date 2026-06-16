package main

import (
	"net"
	"sync"
)

type PeerState struct {
	mu        sync.RWMutex
	static    []net.IP
	alive     map[string]struct{}
	mode      string
	aliveOnly bool
}

func newPeerState(cfg *NodeConfig) *PeerState {
	static := make([]net.IP, 0, len(cfg.Peers))
	for _, p := range cfg.Peers {
		if ip := net.ParseIP(p).To4(); ip != nil {
			static = append(static, ip)
		}
	}
	return &PeerState{
		static:    static,
		alive:     make(map[string]struct{}),
		mode:      cfg.Mode,
		aliveOnly: cfg.AliveOnly,
	}
}

func (ps *PeerState) UpdateAlive(ips []string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.alive = make(map[string]struct{})
	for _, ip := range ips {
		ps.alive[ip] = struct{}{}
	}
}

func (ps *PeerState) Static() []net.IP {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return append([]net.IP(nil), ps.static...)
}

func (ps *PeerState) Serve() []net.IP {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	switch {
	case ps.mode == "static" && !ps.aliveOnly:
		return ps.static

	case ps.mode == "static" && ps.aliveOnly:
		var out []net.IP
		for _, ip := range ps.static {
			if _, ok := ps.alive[ip.String()]; ok {
				out = append(out, ip)
			}
		}
		return out

	case ps.mode == "dynamic" && !ps.aliveOnly:
		seen := make(map[string]struct{})
		var out []net.IP
		for _, ip := range ps.static {
			seen[ip.String()] = struct{}{}
			out = append(out, ip)
		}
		for ipStr := range ps.alive {
			if _, exists := seen[ipStr]; !exists {
				if ip := net.ParseIP(ipStr).To4(); ip != nil {
					out = append(out, ip)
				}
			}
		}
		return out

	default: // dynamic + alive_only
		var out []net.IP
		for ipStr := range ps.alive {
			if ip := net.ParseIP(ipStr).To4(); ip != nil {
				out = append(out, ip)
			}
		}
		return out
	}
}
