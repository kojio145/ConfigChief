package netx

import (
	"encoding/binary"
	"fmt"
	"math/bits"
	"net/netip"
	"strconv"
	"strings"
)

// parse reads "address/length". A bare address counts as a host (/32, /128).
func parse(s string) (netip.Prefix, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return netip.Prefix{}, fmt.Errorf("アドレスが空です")
	}
	if !strings.Contains(s, "/") {
		a, err := netip.ParseAddr(s)
		if err != nil {
			return netip.Prefix{}, fmt.Errorf("「%s」はアドレスとして読めません", s)
		}
		return netip.PrefixFrom(a, a.BitLen()), nil
	}
	p, err := netip.ParsePrefix(s)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("「%s」はアドレス/プレフィックス長として読めません", s)
	}
	return p, nil
}

func parse4(s string) (netip.Prefix, error) {
	p, err := parse(s)
	if err != nil {
		return p, err
	}
	if !p.Addr().Is4() {
		return p, fmt.Errorf("「%s」は IPv4 ではありません", s)
	}
	return p, nil
}

func u32(a netip.Addr) uint32 {
	b := a.As4()
	return binary.BigEndian.Uint32(b[:])
}

func addr4(v uint32) netip.Addr {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	return netip.AddrFrom4(b)
}

func maskBits(n int) uint32 {
	if n <= 0 {
		return 0
	}
	return ^uint32(0) << (32 - n)
}

// IP gives the address part: "192.0.2.1/30" → "192.0.2.1".
func IP(s string) (string, error) {
	p, err := parse(s)
	if err != nil {
		return "", err
	}
	return p.Addr().String(), nil
}

// Mask gives the netmask: "192.0.2.1/30" → "255.255.255.252".
func Mask(s string) (string, error) {
	p, err := parse4(s)
	if err != nil {
		return "", err
	}
	return addr4(maskBits(p.Bits())).String(), nil
}

// Wildcard gives the inverted mask: "192.0.2.1/30" → "0.0.0.3".
func Wildcard(s string) (string, error) {
	p, err := parse4(s)
	if err != nil {
		return "", err
	}
	return addr4(^maskBits(p.Bits())).String(), nil
}

// PrefixLen gives the prefix length: "192.0.2.1/30" → 30.
func PrefixLen(s string) (int, error) {
	p, err := parse(s)
	if err != nil {
		return 0, err
	}
	return p.Bits(), nil
}

// Network gives the network: "192.0.2.1/30" → "192.0.2.0/30".
func Network(s string) (string, error) {
	p, err := parse(s)
	if err != nil {
		return "", err
	}
	return p.Masked().String(), nil
}

// Broadcast gives the last address of the network: "192.0.2.1/30" → "192.0.2.3".
func Broadcast(s string) (string, error) {
	p, err := parse4(s)
	if err != nil {
		return "", err
	}
	return addr4(u32(p.Addr()) | ^maskBits(p.Bits())).String(), nil
}

// Host gives the n-th address of the network, without the length:
// Host("192.0.2.0/30", 2) → "192.0.2.2".
func Host(s string, n int) (string, error) {
	p, err := parse4(s)
	if err != nil {
		return "", err
	}
	size := uint64(1) << (32 - p.Bits())
	if n < 0 || uint64(n) >= size {
		return "", fmt.Errorf("%s に %d 番目のアドレスはありません", p.Masked(), n)
	}
	return addr4(u32(p.Masked().Addr()) + uint32(n)).String(), nil
}

// maskLen gives the prefix length of a contiguous netmask.
func maskLen(mask string) (int, error) {
	m, err := netip.ParseAddr(strings.TrimSpace(mask))
	if err != nil || !m.Is4() {
		return 0, fmt.Errorf("「%s」はマスクとして読めません", mask)
	}
	v := u32(m)
	n := bits.LeadingZeros32(^v)
	if v != maskBits(n) {
		return 0, fmt.Errorf("「%s」は連続したマスクではありません", mask)
	}
	return n, nil
}

// WithMask joins an address and a netmask:
// WithMask("192.0.2.1", "255.255.255.252") → "192.0.2.1/30".
func WithMask(addr, mask string) (string, error) {
	a, err := netip.ParseAddr(strings.TrimSpace(addr))
	if err != nil || !a.Is4() {
		return "", fmt.Errorf("「%s」は IPv4 アドレスとして読めません", addr)
	}
	n, err := maskLen(mask)
	if err != nil {
		return "", err
	}
	return a.String() + "/" + strconv.Itoa(n), nil
}

// WithWildcard joins an address and a wildcard mask:
// WithWildcard("192.0.2.0", "0.0.0.255") → "192.0.2.0/24".
// A wildcard that is not the inverse of a contiguous mask is an error.
func WithWildcard(addr, wildcard string) (string, error) {
	a, err := netip.ParseAddr(strings.TrimSpace(addr))
	if err != nil || !a.Is4() {
		return "", fmt.Errorf("「%s」は IPv4 アドレスとして読めません", addr)
	}
	w, err := netip.ParseAddr(strings.TrimSpace(wildcard))
	if err != nil || !w.Is4() {
		return "", fmt.Errorf("「%s」はワイルドカードとして読めません", wildcard)
	}
	v := ^u32(w)
	n := bits.LeadingZeros32(^v)
	if v != maskBits(n) {
		return "", fmt.Errorf("「%s」は連続したワイルドカードではありません", wildcard)
	}
	return a.String() + "/" + strconv.Itoa(n), nil
}

// InNet tells whether the address (its length is ignored) is inside the network.
func InNet(addr, network string) (bool, error) {
	a, err := parse(addr)
	if err != nil {
		return false, err
	}
	n, err := parse(network)
	if err != nil {
		return false, err
	}
	return n.Masked().Contains(a.Addr()), nil
}
