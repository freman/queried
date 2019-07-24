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
	"os"

	gct "github.com/freman/go-commontypes"
	"github.com/naoina/toml"
)

type forwardedZone struct {
	Name              string
	Authoritative     bool
	Upstream          string
	Private           bool
	Override          map[string]gct.IP
	NonLocalOverride  map[string]gct.IP
	OverrideResponses bool
}

var config = struct {
	Resolvers     []string
	Listen        []string
	LocalNetworks gct.Networks
	ForwardedZone []forwardedZone
}{
	Resolvers:     []string{},
	Listen:        []string{},
	LocalNetworks: gct.Networks{},
	ForwardedZone: []forwardedZone{},
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
