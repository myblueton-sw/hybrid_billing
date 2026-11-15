package trust

import "net/netip"

func prefixes(list ...string) []netip.Prefix {
	out := make([]netip.Prefix, len(list))
	for index, text := range list {
		out[index] = netip.MustParsePrefix(text)
	}
	return out
}

var (
	metadata = []netip.Addr{netip.MustParseAddr("192.0.0.192"), netip.MustParseAddr("100.100.100.200"), netip.MustParseAddr("168.63.129.16"),
		netip.MustParseAddr("fd00:ec2::254"), netip.MustParseAddr("fd20:ce::254")}
	linkLocal = prefixes("169.254.0.0/16", "fe80::/10")
	forbidden = prefixes("0.0.0.0/8", "127.0.0.0/8", "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
		"224.0.0.0/4", "240.0.0.0/4", "::/96", "::ffff:0:0:0/96", "64:ff9b::/96", "64:ff9b:1::/48", "100::/64", "2001::/23", "2001:db8::/32", "3fff::/20",
		"2002::/16", "5f00::/16", "fec0::/10", "ff00::/8")
	private = prefixes("10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10", "fc00::/7")
	global  = netip.MustParsePrefix("2000::/3")
)

func within(address netip.Addr, list []netip.Prefix) bool {
	for _, prefix := range list {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func classify(address netip.Addr, allowed []netip.Prefix) string {
	if !address.IsValid() || address.Zone() != "" {
		return ReasonForbiddenAddress
	}
	address = address.Unmap()
	if within(address, linkLocal) {
		return ReasonMetadataAddress
	}
	for _, known := range metadata {
		if address == known {
			return ReasonMetadataAddress
		}
	}
	if within(address, forbidden) {
		return ReasonForbiddenAddress
	}
	if within(address, private) {
		if within(address, allowed) {
			return ""
		}
		return ReasonPrivateAddress
	}
	if address.Is6() && !global.Contains(address) {
		return ReasonForbiddenAddress
	}
	return ""
}

func allowable(prefix netip.Prefix) bool {
	if !prefix.IsValid() || prefix != prefix.Masked() {
		return false
	}
	for _, area := range private {
		if area.Bits() <= prefix.Bits() && area.Contains(prefix.Addr()) {
			return true
		}
	}
	return false
}
