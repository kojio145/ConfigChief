package ops

import "testing"

// The IPv6 destinations of the management and of the line monitoring go
// from one vendor to the other, both ways (appendix A 3.23, 2026-11-21): a
// name server, syslog, SNMP and NTP over IPv6, and a watch by ping of an
// IPv6 address. Each vendor takes the source and the output interface of
// the watch its own way: the packs write them so and tell.
func TestCrossVendorIPv6Destinations(t *testing.T) {
	crossSites(t, "ipv6_dest", []site{
		{
			old: "rt-w01.txt", oldHost: "rt-w01", newHost: "ix-w01", pack: "nec_ix",
			ifs:   map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0"},
			// Nothing to mend: the source of the probe becomes the address
			// of its interface, and its output interface the one of the
			// route to the target.
			first: 3,
			notes: []string{"「IPv6 の転送」を有効にする行は NEC IX にありません", "IPv6 アドレス 2001:db8:0:2::1 にして書きます", "出力 IF GigaEthernet1.0 にして書きます"},
		},
		{
			old: "ix-w01.txt", oldHost: "ix-w01", newHost: "rt-w01", pack: "cisco_ios",
			ifs:   map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1"},
			// Nothing to mend: the named watch gets a number, the source
			// address becomes its interface, and the output interface (the
			// one of the route anyway) is not written.
			first: 4,
			notes: []string{"「IPv6 の転送」を有効にして書きます", "1 番として書きます", "IF GigabitEthernet0/1 にして書きます", "「監視の出口の IF」GigabitEthernet0/1 は書きません"},
		},
	})
}
