// Package bulk makes up the configurations of a large case: many devices of
// about the same size, to see how long the steps take (spec 16 and 19).
//
// Every value is made up from the number of the device, so the same call
// gives the same files. The addresses are of the block kept for tests of
// network gear (198.18.0.0/15, appendix B 11.1), the AS numbers of the ones
// kept for documents.
//
// The devices come in pairs, one site each: the first of a pair is written
// as Cisco IOS, the second as NEC IX. The two share the LANs of the site
// (VRRP and OSPF between them), and each has a line of its own to a
// provider that is not part of the case.
package bulk

import (
	"fmt"
	"strings"
)

// File is one configuration.
type File struct {
	Name string // the name of the file, "<host>.txt"
	Host string
	Pack string // the pack that reads it
	Text string
}

// Most is the largest number of devices Configs makes (the addresses of the
// LANs run out after it).
const Most = 500

// The least a device is made of, whatever the number of lines asked for.
const (
	vlans    = 8   // the LANs of a site
	filters  = 120 // the lines of the access list
	prefixes = 80  // the lines of the prefix list
)

// Configs makes the configurations of n devices of the given number of
// lines each. The static routes fill a device up to that number; a device
// never has fewer lines than its fixed parts need (about 345).
func Configs(n, lines int) ([]File, error) {
	if n < 1 || n > Most {
		return nil, fmt.Errorf("bulk: the number of devices is 1 to %d, not %d", Most, n)
	}
	out := make([]File, 0, n)
	for i := 0; i < n; i++ {
		d := device{i: i, site: i / 2, second: i%2 == 1}
		write := d.cisco
		pack := "cisco_ios"
		if d.second {
			write, pack = d.nec, "nec_ix"
		}
		// One route first, to learn how many lines the rest takes.
		text := write(1)
		if more := lines - strings.Count(text, "\n"); more > 0 {
			text = write(1 + more)
		}
		out = append(out, File{Name: d.host() + ".txt", Host: d.host(), Pack: pack, Text: text})
	}
	return out, nil
}

type device struct {
	i, site int
	second  bool
}

func (d device) host() string {
	if d.second {
		return fmt.Sprintf("ix-b%03db", d.site+1)
	}
	return fmt.Sprintf("rt-b%03da", d.site+1)
}

// loopback is the address that names the device.
func (d device) loopback() string {
	return fmt.Sprintf("198.18.%d.%d", d.i/250, d.i%250+1)
}

// wan is the two ends of the line to the provider: this device, the provider.
func (d device) wan() (self, peer string) {
	base := 4 * d.i
	return fmt.Sprintf("198.18.%d.%d", 8+base/256, base%256+1), fmt.Sprintf("198.18.%d.%d", 8+base/256, base%256+2)
}

// subnet is LAN v of the site (a /27): the network, the address the two
// devices share, the address of this device.
func (d device) subnet(v int) (network, shared, self string) {
	at := func(host int) string { return fmt.Sprintf("198.19.%d.%d", d.site, v*32+host) }
	if d.second {
		return at(0), at(1), at(3)
	}
	return at(0), at(1), at(2)
}

// far is network j behind the provider (a /28).
func far(j int) string {
	j %= 1024
	return fmt.Sprintf("198.18.%d.%d", 64+j/16, j%16*16)
}

const (
	nameServer = "198.18.2.53"
	timeServer = "198.18.2.123"
	logServer  = "198.18.2.50"
	trapServer = "198.18.2.60"
	accessList = "acl-b-in"
	prefixList = "pl-b-in"
)

func (d device) priority() int {
	if d.second {
		return 100
	}
	return 110
}

// cisco writes the device as Cisco IOS with the given number of routes
// beside the default route.
func (d device) cisco(routes int) string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	self, peer := d.wan()
	lo := d.loopback()

	p("hostname %s", d.host())
	p("!")
	p("ip name-server %s", nameServer)
	p("!")
	p("interface Loopback0")
	p(" description sample id")
	p(" ip address %s 255.255.255.255", lo)
	p("!")
	p("interface GigabitEthernet0/0")
	p(" description sample line %d", d.i+1)
	p(" ip address %s 255.255.255.252", self)
	p(" ip access-group %s in", accessList)
	p(" no shutdown")
	p("!")
	p("interface GigabitEthernet0/1")
	p(" no shutdown")
	p("!")
	for v := 0; v < vlans; v++ {
		_, shared, own := d.subnet(v)
		p("interface GigabitEthernet0/1.%d", 101+v)
		p(" description sample lan %d", v+1)
		p(" encapsulation dot1Q %d", 101+v)
		p(" ip address %s 255.255.255.224", own)
		p(" vrrp %d ip %s", v+1, shared)
		p(" vrrp %d priority %d", v+1, d.priority())
		p(" no shutdown")
		p("!")
	}
	p("router ospf 1")
	p(" router-id %s", lo)
	for v := 1; v < vlans; v++ {
		p(" passive-interface GigabitEthernet0/1.%d", 101+v)
	}
	p(" network %s 0.0.0.0 area 0", lo)
	for v := 0; v < vlans; v++ {
		network, _, _ := d.subnet(v)
		p(" network %s 0.0.0.31 area 0", network)
	}
	p("!")
	p("router bgp 64500")
	p(" bgp router-id %s", lo)
	p(" neighbor %s remote-as 64501", peer)
	p(" neighbor %s description sample-provider", peer)
	p(" neighbor %s prefix-list %s in", peer, prefixList)
	for v := 0; v < vlans; v++ {
		network, _, _ := d.subnet(v)
		p(" network %s mask 255.255.255.224", network)
	}
	p("!")
	p("ip route 0.0.0.0 0.0.0.0 %s", peer)
	for j := 0; j < routes; j++ {
		p("ip route %s 255.255.255.240 %s", far(j), peer)
	}
	p("!")
	p("ip access-list extended %s", accessList)
	for j := 0; j < filters; j++ {
		network, _, _ := d.subnet(j % vlans)
		p(" permit tcp %s 0.0.0.15 %s 0.0.0.31 eq %d", far(j), network, 1024+j)
	}
	p("!")
	for j := 0; j < prefixes; j++ {
		p("ip prefix-list %s seq %d permit %s/28", prefixList, 10*(j+1), far(j))
	}
	p("!")
	p("logging host %s", logServer)
	p("!")
	p("snmp-server community dummy-ro-comm RO")
	p("snmp-server host %s version 2c dummy-ro-comm", trapServer)
	p("!")
	p("ntp server %s", timeServer)
	p("!")
	p("line vty 0 4")
	p(" exec-timeout 10 0")
	p(" login local")
	p(" transport input ssh")
	p("!")
	p("end")
	return b.String()
}

// nec writes the device as NEC IX with the given number of routes beside
// the default route.
func (d device) nec(routes int) string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format+"\n", a...) }
	self, peer := d.wan()
	lo := d.loopback()

	p("hostname %s", d.host())
	p("!")
	for j := 0; j < filters; j++ {
		network, _, _ := d.subnet(j % vlans)
		p("ip access-list %s permit tcp src %s/28 dest %s/27 dport eq %d", accessList, far(j), network, 1024+j)
	}
	for j := 0; j < prefixes; j++ {
		p("ip prefix-list %s %d permit %s/28", prefixList, 10*(j+1), far(j))
	}
	p("!")
	p("ip name-server %s", nameServer)
	p("!")
	p("ip route default %s", peer)
	for j := 0; j < routes; j++ {
		p("ip route %s/28 %s", far(j), peer)
	}
	p("!")
	p("ip router ospf 1")
	p("  router-id %s", lo)
	for v := 1; v < vlans; v++ {
		p("  passive-interface GigaEthernet1.%d", v+1)
	}
	for v := 0; v < vlans; v++ {
		p("  network GigaEthernet1.%d area 0", v+1)
	}
	p("  network Loopback0.0 area 0")
	p("!")
	p("router bgp 64500")
	p("  router-id %s", lo)
	p("  neighbor %s remote-as 64501", peer)
	p("  neighbor %s description sample-provider", peer)
	p("  address-family ipv4 unicast")
	for v := 0; v < vlans; v++ {
		network, _, _ := d.subnet(v)
		p("    network %s/27", network)
	}
	p("    neighbor %s distribute-list %s in", peer, prefixList)
	p("!")
	p("snmp-agent ip enable")
	p("snmp-agent ip community dummy-ro-comm ro")
	p("snmp-agent ip host %s dummy-ro-comm version 2", trapServer)
	p("!")
	p("ntp server %s", timeServer)
	p("!")
	p("syslog ip host %s", logServer)
	p("!")
	p("ssh-server ip enable")
	p("terminal timeout 10")
	p("!")
	p("vrrp enable")
	p("!")
	p("interface GigaEthernet0.0")
	p("  description sample line %d", d.i+1)
	p("  ip address %s/30", self)
	p("  ip filter %s 1 in", accessList)
	p("  no shutdown")
	p("!")
	p("interface GigaEthernet1.0")
	p("  no shutdown")
	p("!")
	for v := 0; v < vlans; v++ {
		_, shared, own := d.subnet(v)
		p("interface GigaEthernet1.%d", v+1)
		p("  description sample lan %d", v+1)
		p("  encapsulation dot1q %d", 101+v)
		p("  ip address %s/27", own)
		p("  vrrp %d ip %s", v+1, shared)
		p("  vrrp %d priority %d", v+1, d.priority())
		p("  no shutdown")
		p("!")
	}
	p("interface Loopback0.0")
	p("  description sample id")
	p("  ip address %s/32", lo)
	p("!")
	return b.String()
}
