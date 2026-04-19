package main

import (
	"fmt"
	"log"
	"time"

	"github.com/miekg/dns"
)

func runDNS(cfg *Config, ps *PeerState) {
	origin := dns.Fqdn(cfg.DNS.Origin)
	addr := fmt.Sprintf("%s:%d", cfg.DNS.Host, cfg.DNS.Port)

	mux := dns.NewServeMux()
	mux.HandleFunc(origin, func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.RecursionAvailable = false

		if len(r.Question) == 0 {
			w.WriteMsg(m)
			return
		}

		q := r.Question[0]

		switch q.Qtype {
		case dns.TypeA:
			for _, ip := range ps.Serve() {
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

		default:
			m.SetRcode(r, dns.RcodeNameError)
		}

		// empty answer — add SOA to authority so resolvers know we're authoritative
		if len(m.Answer) == 0 && m.Rcode == dns.RcodeSuccess {
			m.Ns = append(m.Ns, buildSOA(cfg))
		}

		w.WriteMsg(m)
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
