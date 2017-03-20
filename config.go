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
	"io/ioutil"
	"net"
	"os"

	gct "github.com/freman/go-commontypes"
	"github.com/naoina/toml"
)

type forwardedZone struct {
	Name          string
	Authoritative bool
	Upstream      string
	Private       bool
	Override      map[string]gct.IP
}

var config = struct {
	Resolvers     []string
	Listen        []string
	LocalNetworks gct.Networks
	ForwardedZone []forwardedZone
}{
	Resolvers: []string{"8.8.8.8", "8.8.4.4"},
	Listen:    []string{":53"},
	LocalNetworks: gct.Networks{
		gct.Network{IPNet: &net.IPNet{IP: []byte{252, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Mask: []byte{254, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}}},
		gct.Network{IPNet: &net.IPNet{IP: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, Mask: []byte{255, 255, 255, 255, 255, 255, 255, 255, 0, 0, 0, 0, 0, 0, 0, 0}}},
		gct.Network{IPNet: &net.IPNet{IP: []byte{127, 0, 0, 0}, Mask: []byte{255, 0, 0, 0}}},
		gct.Network{IPNet: &net.IPNet{IP: []byte{10, 0, 0, 0}, Mask: []byte{255, 0, 0, 0}}},
		gct.Network{IPNet: &net.IPNet{IP: []byte{172, 16, 0, 0}, Mask: []byte{255, 240, 0, 0}}},
		gct.Network{IPNet: &net.IPNet{IP: []byte{192, 168, 0, 0}, Mask: []byte{255, 255, 0, 0}}},
	},
	ForwardedZone: []forwardedZone{
		forwardedZone{
			Name:          "consul.",
			Authoritative: true,
			Upstream:      "172.31.1.2:8600",
			Private:       true,
		},
		forwardedZone{
			Name:          "some.example.com.",
			Authoritative: true,
			Upstream:      "10.23.2.2:53",
			Private:       false,
		},
	},
}

func loadConfig(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	if err := toml.Unmarshal(buf, &config); err != nil {
		return err
	}
	return nil
}
