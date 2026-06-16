package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/miekg/dns"
)

func runDNS(cfg *Config, ps *PeerState) {
	origin := dns.Fqdn(cfg.DNS.Origin)
	addr := fmt.Sprintf("%s:%d", cfg.DNS.Host, cfg.DNS.Port)

	mux := dns.NewServeMux()
	mux.HandleFunc(origin, func(w dns.ResponseWriter, r *dns.Msg) {
		w.WriteMsg(handleDNSQuery(cfg, ps, origin, r))
	})

	udp := &dns.Server{Addr: addr, Net: "udp", Handler: mux}
	tcp := &dns.Server{Addr: addr, Net: "tcp", Handler: mux}

	log.Printf("dns: listening on %s (UDP+TCP)", addr)

	go func() {
		if err := tcp.ListenAndServe(); err != nil {
			log.Fatalf("dns: TCP server error: %v", err)
		}
	}()

	if err := udp.ListenAndServe(); err != nil {
		log.Fatalf("dns: UDP server error: %v", err)
	}
}

func handleDNSQuery(cfg *Config, ps *PeerState, origin string, r *dns.Msg) *dns.Msg {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.RecursionAvailable = false

	if len(r.Question) == 0 {
		return m
	}

	q := r.Question[0]
	if !dns.IsSubDomain(origin, q.Name) || !equalName(origin, q.Name) {
		m.SetRcode(r, dns.RcodeNameError)
		return m
	}

	switch q.Qtype {
	case dns.TypeA:
		for _, ip := range selectIPs(ps.Serve(), cfg.DNS.MaxRecords) {
			m.Answer = append(m.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    60,
				},
				A: ip,
			})
		}

	case dns.TypeNS:
		m.Answer = append(m.Answer, &dns.NS{
			Hdr: dns.RR_Header{
				Name:   origin,
				Rrtype: dns.TypeNS,
				Class:  dns.ClassINET,
				Ttl:    3600,
			},
			Ns: dns.Fqdn(cfg.DNS.NS),
		})

	case dns.TypeSOA:
		m.Answer = append(m.Answer, buildSOA(cfg))
	}

	// Empty answers at an existing name are NODATA, not NXDOMAIN. Include SOA
	// so recursive resolvers can cache the negative answer correctly.
	if len(m.Answer) == 0 && m.Rcode == dns.RcodeSuccess {
		m.Ns = append(m.Ns, buildSOA(cfg))
	}

	return m
}

func equalName(a, b string) bool {
	return dns.CanonicalName(a) == dns.CanonicalName(b)
}

func selectIPs(ips []net.IP, maxRecords int) []net.IP {
	if maxRecords == 0 || len(ips) <= maxRecords {
		return ips
	}

	selected := append([]net.IP(nil), ips...)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(selected), func(i, j int) {
		selected[i], selected[j] = selected[j], selected[i]
	})
	return selected[:maxRecords]
}

func buildSOA(cfg *Config) *dns.SOA {
	return &dns.SOA{
		Hdr: dns.RR_Header{
			Name:   dns.Fqdn(cfg.DNS.Origin),
			Rrtype: dns.TypeSOA,
			Class:  dns.ClassINET,
			Ttl:    3600,
		},
		Ns:      dns.Fqdn(cfg.DNS.NS),
		Mbox:    dns.Fqdn(cfg.DNS.Email),
		Serial:  uint32(time.Now().Unix()),
		Refresh: 3600,
		Retry:   900,
		Expire:  604800,
		Minttl:  300,
	}
}
