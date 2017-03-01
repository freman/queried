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
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/miekg/dns"
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

				l.Fatal("Server exited unexpectantly")
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

func zoneHandler(zone forwardedZone) func(dns.ResponseWriter, *dns.Msg) {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		proto, ip := netip(w)
		if zone.Private && !config.LocalNetworks.Contains(ip) {
			return
		}

		c := new(dns.Client)
		c.Net = proto
		in, _, err := c.Exchange(r, zone.Upstream)
		if err != nil {
			return
		}
		if zone.Authoritative {
			in.Authoritative = true
			in.MsgHdr.Authoritative = true
		}
		w.WriteMsg(in)
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
