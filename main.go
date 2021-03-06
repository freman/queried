/*
 *   queried - forward dns with authority while still recursivly resolving
 *   Copyright (c) 2017 Shannon Wynter.
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	gct "github.com/freman/go-commontypes"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

var (
	version = "Undefined"
	commit  = "Undefined"
)

func init() {
	rand.Seed(time.Now().Unix())
}

func main() {
	configFile := flag.String("config", "config.toml", "Config file")
	debug := flag.Bool("debug", false, "Debug log level")
	showVersion := flag.Bool("version", false, "Show version and exit")

	flag.Parse()

	if *showVersion {
		fmt.Printf("queried - %s (%s)\n", version, commit)
		fmt.Println("https://github.com/freman/queried")
		return
	}

	if *debug {
		log.SetLevel(log.DebugLevel)
	}

	if *configFile != "" {
		if err := loadConfig(*configFile); err != nil {
			log.WithField("configFile", *configFile).WithError(err).Fatal("Unable to load config file")
		}
	}

	if *debug {
		log.WithField("config", config).Debug("Parsed configuration")
	}

	for _, zone := range config.ForwardedZone {
		dns.HandleFunc(zone.Name, zoneHandler(zone))
	}

	dns.HandleFunc(".", handleDefault)

	for _, listen := range config.Listen {
		for _, proto := range []string{"udp", "tcp"} {
			go func(listen, proto string) {
				l := log.WithFields(log.Fields{
					"listen": listen,
					"proto":  proto,
				})
				server := &dns.Server{Addr: listen, Net: proto}
				if err := server.ListenAndServe(); err != nil {
					l.WithError(err).Fatal("Unable to listen")
				}

				l.Fatal("Server exited expectantly")
			}(listen, proto)
		}
	}

	ch := make(chan bool, 1)
	<-ch
}

func netip(w dns.ResponseWriter) (proto string, realIP net.IP) {
	if addr, ok := w.RemoteAddr().(*net.UDPAddr); ok {
		realIP = make(net.IP, len(addr.IP))
		copy(realIP, addr.IP)
		proto = "udp"
	} else if addr, ok := w.RemoteAddr().(*net.TCPAddr); ok {
		realIP = make(net.IP, len(addr.IP))
		copy(realIP, addr.IP)
		proto = "tcp"
	}
	return
}

func searchOverride(haystack map[string]gct.IP, needle string) *net.IP {
	logCtx := log.WithFields(log.Fields{
		"haystack": haystack,
		"needle":   needle,
	})
	logCtx.Debug("Searching...")

	// Quick search first for a simple match
	if ip, found := haystack[needle]; found {
		logCtx.Debug("Direct match")
		return &ip.IP
	}

	// Long slow slog through all the overrides for wildcards
	for name, ip := range haystack {
		if !strings.HasPrefix(name, "*.") {
			continue
		}
		check := strings.TrimPrefix(name, "*")
		if strings.HasSuffix(needle, check) {
			logCtx.Debug("Wildcard match")
			return &ip.IP
		}
	}
	return nil
}

func zoneHandler(zone forwardedZone) func(dns.ResponseWriter, *dns.Msg) {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		proto, ip := netip(w)
		localRequest := config.LocalNetworks.Contains(ip)
		if zone.Private && !localRequest {
			return
		}

		var overridden bool

		reply := new(dns.Msg)
		reply.SetReply(r)

		zoneSuffix := "." + zone.Name
		for _, q := range r.Question {
			if q.Qtype == dns.TypeA && strings.HasSuffix(q.Name, zoneSuffix) {
				domain := strings.TrimSuffix(q.Name, zoneSuffix)
				if !localRequest {
					if ip := searchOverride(zone.NonLocalOverride, domain); ip != nil {
						log.WithFields(log.Fields{
							"question": r.Question[0],
							"ip":       ip,
						}).Debug("Found an IP")
						reply.Answer = append(reply.Answer, &dns.A{
							Hdr: dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: q.Qclass, Ttl: 60},
							A:   *ip,
						})
						overridden = true
						continue
					}
				}
				if ip := searchOverride(zone.Override, domain); ip != nil {
					log.WithFields(log.Fields{
						"question": r.Question[0],
						"ip":       ip,
					}).Debug("Found an IP")
					reply.Answer = append(reply.Answer, &dns.A{
						Hdr: dns.RR_Header{Name: q.Name, Rrtype: q.Qtype, Class: q.Qclass, Ttl: 60},
						A:   *ip,
					})
					overridden = true

				}
			}
		}

		if len(reply.Answer) == 0 {
			c := new(dns.Client)
			c.Net = proto

			var err error
			reply, _, err = c.Exchange(r, zone.Upstream)
			if err != nil {
				return
			}
		}

		if zone.Authoritative {
			reply.Authoritative = true
			reply.MsgHdr.Authoritative = true
		}

		if zone.OverrideResponses && !overridden {
			for _, field := range []*[]dns.RR{&reply.Answer, &reply.Extra} {
				for i, rr := range *field {
					if a, isa := rr.(*dns.A); isa {
						hostPart := strings.TrimSuffix(a.Hdr.Name, zoneSuffix)
						if ip := searchOverride(zone.Override, hostPart); ip != nil {
							a.A = *ip
							(*field)[i] = a
							continue
						}
						if !localRequest {
							if ip := searchOverride(zone.NonLocalOverride, hostPart); ip != nil {
								a.A = *ip
								(*field)[i] = a
							}
						}
					}
				}
			}
		}

		w.WriteMsg(reply)
	}
}

func handleDefault(w dns.ResponseWriter, r *dns.Msg) {
	proto, ip := netip(w)
	if !config.LocalNetworks.Contains(ip) {
		return
	}

	for _, i := range rand.Perm(len(config.Resolvers)) {
		c := new(dns.Client)
		c.Net = proto
		in, _, err := c.Exchange(r, config.Resolvers[i])
		if err == nil {
			w.WriteMsg(in)
			break
		}
	}
}
