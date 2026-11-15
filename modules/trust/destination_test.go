package trust

import (
	"net/netip"
	"strings"
	"testing"
)

func (f world) destination(id, raw string, private ...string) DestinationView {
	f.t.Helper()
	var list []netip.Prefix
	for _, text := range private {
		list = append(list, netip.MustParsePrefix(text))
	}
	_, err := f.r.ProposeDestination("w1", Destination{ID: id, URL: raw, Private: list}, ann)
	f.must(err)
	view, err := f.r.ApproveDestination("w1", id, bob)
	f.must(err)
	return view
}

func TestCanonicalURLs(t *testing.T) {
	good := map[string]string{
		"https://Example.COM":                         "https://example.com:443",
		"https://example.com.":                        "https://example.com:443",
		"https://example.com:443/":                    "https://example.com:443/",
		"https://example.com:8443/hook/a%20b":         "https://example.com:8443/hook/a%20b",
		"https://xn--bcher-kva.example":               "https://xn--bcher-kva.example:443",
		"https://8.8.8.8":                             "https://8.8.8.8:443",
		"https://[2001:4860:4860:0:0:0:0:8888]:9000":  "https://[2001:4860:4860::8888]:9000",
		"https://a-b.c0":                              "https://a-b.c0:443",
		"https://" + strings.Repeat("a", 63) + ".com": "https://" + strings.Repeat("a", 63) + ".com:443",
		"https://example.com:65535":                   "https://example.com:65535",
		"https://example.com:1":                       "https://example.com:1",
	}
	for raw, want := range good {
		if got, ok := parse(raw, false); !ok || got.url != want {
			t.Errorf("%s: %+v %v; want %s", raw, got, ok, want)
		}
	}
	for _, raw := range []string{
		"", "http://example.com", "https://user@example.com", "https://example.com/?q", "https://example.com/?", "https://example.com/#f", "https:example.com",
		"https://", "https://example.com:", "https://example.com:0", "https://example.com:0443", "https://example.com:65536", "https://example.com:x",
		"https://bücher.example", "https://2130706433", "https://0x7f.1", "https://127.1", "https://1.2.3", "https://-a.com", "https://a-.com", "https://a..com",
		"https://" + strings.Repeat("a", 64) + ".com", "https://" + strings.Repeat("a.", 127) + "com", "https://exa mple.com", "https://example.com\\@evil",
		"https://example.com/\x01", "https://[fe80::1%25eth0]", "https://[1.2.3.4]", "https://[::ffff:8.8.8.8]", "https://%E2%84%AAex.com", "https://ex%41mple.com", "https://[::ffff:1.2.3.4", "https://ex_ample.com", "https://_a.example", "https://a._b", "https://.", "%zz",
		"https://example.com/" + strings.Repeat("a", maxLocation),
	} {
		if got, ok := parse(raw, false); ok {
			t.Errorf("%q accepted as %+v", raw, got)
		}
	}
	if got, ok := parse("https://example.com/a?b=c#d", true); !ok || got.url != "https://example.com:443/a?b=c" {
		t.Errorf("a redirect form %+v %v", got, ok)
	}
	if got, ok := parse("https://example.com/a?", true); !ok || got.url != "https://example.com:443/a" {
		t.Errorf("an empty query %+v %v", got, ok)
	}
	if _, ok := dnsHost(strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 61)); !ok {
		t.Errorf("a host of 253 bytes is refused")
	}
	if _, ok := dnsHost(strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 62)); ok {
		t.Errorf("a host of 254 bytes is accepted")
	}
	for _, host := range []string{"a.z", "a.a1", "z9.example"} {
		if _, ok := dnsHost(host); !ok {
			t.Errorf("%s is refused", host)
		}
	}
	for _, host := range []string{"a.9z", "a.-z", "A.{", "a.`b"} {
		if _, ok := dnsHost(host); ok {
			t.Errorf("%s is accepted", host)
		}
	}
}

func TestDestinationLifecycle(t *testing.T) {
	f := setup(t)
	expect(t, "no workspace", second(f.r.ProposeDestination("", Destination{ID: "d1", URL: "https://a.com"}, ann)), ErrInvalid)
	expect(t, "no identifier", second(f.r.ProposeDestination("w1", Destination{URL: "https://a.com"}, ann)), ErrInvalid)
	expect(t, "an invalid url", second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "http://a.com"}, ann)), ErrInvalid)
	for name, list := range map[string][]string{
		"too many prefixes":      {"10.0.0.0/8", "10.1.0.0/16", "10.2.0.0/16"},
		"a repeated prefix":      {"10.0.0.0/8", "10.0.0.0/8"},
		"an unmasked prefix":     {"10.1.2.3/8"},
		"a public prefix":        {"8.8.0.0/16"},
		"everything":             {"0.0.0.0/0"},
		"all of IPv6":            {"::/0"},
		"a mapped prefix":        {"::ffff:10.0.0.0/104"},
		"a wider private prefix": {"172.0.0.0/8"},
		"a loopback prefix":      {"127.0.0.0/8"},
		"a metadata range":       {"169.254.0.0/16"},
	} {
		var prefixes []netip.Prefix
		for _, text := range list {
			prefixes = append(prefixes, netip.MustParsePrefix(text))
		}
		expect(t, name, second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://a.com", Private: prefixes}, ann)), ErrInvalid)
	}
	expect(t, "an invalid prefix", second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://a.com", Private: []netip.Prefix{{}}}, ann)), ErrInvalid)
	reason(t, "an assistant proposing", second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://a.com"}, bot)), ReasonAINotAllowed)
	for _, raw := range []string{"https://127.0.0.1", "https://169.254.169.254", "https://[::1]", "https://10.0.0.1"} {
		var refusal Refusal
		if err := second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: raw}, ann)); !errorsAs(err, &refusal) {
			t.Errorf("%s: %v", raw, err)
		}
	}
	allowed := netip.MustParsePrefix("10.1.0.0/16")
	view, err := f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://10.1.2.3:8443/hook", Private: []netip.Prefix{allowed, netip.MustParsePrefix("fd12::/16")}}, ann)
	if err != nil || view.State != StateProposed || view.Destination.URL != "https://10.1.2.3:8443/hook" || view.Destination.Private[0].String() != "10.1.0.0/16" {
		t.Fatalf("a private literal destination %+v %v", view, err)
	}
	view.Destination.Private[0] = netip.MustParsePrefix("10.9.0.0/16")
	if again, err := f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://10.1.2.3:8443/hook", Private: []netip.Prefix{netip.MustParsePrefix("fd12::/16"), allowed}}, ann); err != nil || again.Destination.Private[0] != allowed {
		t.Errorf("the repeated proposal %+v %v", again, err)
	}
	expect(t, "another proposer", second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://10.1.2.3:8443/hook", Private: []netip.Prefix{allowed, netip.MustParsePrefix("fd12::/16")}}, bob)), ErrConflict)
	expect(t, "another url", second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://10.1.2.3:8443/other", Private: []netip.Prefix{allowed, netip.MustParsePrefix("fd12::/16")}}, ann)), ErrConflict)
	expect(t, "another allowlist", second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://10.1.2.3:8443/hook", Private: []netip.Prefix{allowed}}, ann)), ErrConflict)
	expect(t, "another prefix of the same count", second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://10.1.2.3:8443/hook", Private: []netip.Prefix{allowed, netip.MustParsePrefix("fd13::/16")}}, ann)), ErrConflict)
	expect(t, "revoking an unknown destination", second(f.r.RevokeDestination("w1", "d9", ann)), ErrNotFound)
	expect(t, "the same origin under another identifier", second(f.r.ProposeDestination("w1", Destination{ID: "d2", URL: "https://10.1.2.3:8443/else", Private: []netip.Prefix{allowed}}, ann)), ErrConflict)
	for _, call := range []func() error{
		func() error { return second(f.r.ApproveDestination("w1", "", bob)) }, func() error { return second(f.r.RevokeDestination("", "d1", bob)) },
		func() error { return second(f.r.DestinationOf("w1", "")) }, func() error { return second(f.r.DestinationOf("", "d1")) },
	} {
		expect(t, "an invalid call", call(), ErrInvalid)
	}
	reason(t, "an assistant approving", second(f.r.ApproveDestination("w1", "d1", bot)), ReasonAINotAllowed)
	reason(t, "an assistant revoking", second(f.r.RevokeDestination("w1", "d1", bot)), ReasonAINotAllowed)
	expect(t, "an unknown destination", second(f.r.ApproveDestination("w1", "d9", bob)), ErrNotFound)
	expect(t, "an unknown workspace", second(f.r.DestinationOf("w9", "d1")), ErrNotFound)
	reason(t, "the proposer approving", second(f.r.ApproveDestination("w1", "d1", ann)), ReasonSelfApproval)
	reason(t, "checking a proposed destination", f.r.CheckAddress("w1", "d1", netip.MustParseAddrPort("10.1.2.3:8443")), ReasonNotActive)
	if active, err := f.r.ApproveDestination("w1", "d1", bob); err != nil || active.State != StateActive || active.Approver != "bob" {
		t.Errorf("approval %+v %v", active, err)
	}
	f.must(second(f.r.ApproveDestination("w1", "d1", bob)))
	reason(t, "another approver", second(f.r.ApproveDestination("w1", "d1", cat)), ReasonNotPending)
	f.must(f.r.CheckAddress("w1", "d1", netip.MustParseAddrPort("10.1.2.3:8443")))
	f.must(f.r.CheckAddress("w1", "d1", netip.MustParseAddrPort("[::ffff:10.1.2.3]:8443")))
	reason(t, "another address than the literal", f.r.CheckAddress("w1", "d1", netip.MustParseAddrPort("10.1.2.4:8443")), ReasonAddressMismatch)
	expect(t, "an active destination proposed again", second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://10.1.2.3:8443/hook", Private: []netip.Prefix{allowed, netip.MustParsePrefix("fd12::/16")}}, ann)), ErrConflict)
	if gone, err := f.r.RevokeDestination("w1", "d1", cat); err != nil || gone.State != StateRevoked {
		t.Errorf("revocation %+v %v", gone, err)
	}
	reason(t, "checking a revoked destination", f.r.CheckAddress("w1", "d1", netip.MustParseAddrPort("10.1.2.3:8443")), ReasonNotActive)
	reason(t, "reusing a revoked identifier", second(f.r.ProposeDestination("w1", Destination{ID: "d1", URL: "https://b.com"}, ann)), ReasonRevoked)
	f.must(second(f.r.ProposeDestination("w1", Destination{ID: "d2", URL: "https://10.1.2.3:8443/else", Private: []netip.Prefix{allowed}}, ann)))
	if read, err := f.r.DestinationOf("w1", "d2"); err != nil || read.State != StateProposed {
		t.Errorf("reading %+v %v", read, err)
	}
	l := limits
	l.Destinations, l.Workspaces = 1, 1
	g := setupWith(t, l)
	g.destination("d1", "https://a.com")
	expect(t, "a second destination", second(g.r.ProposeDestination("w1", Destination{ID: "d2", URL: "https://b.com"}, ann)), ErrLimit)
	expect(t, "a second workspace", second(g.r.ProposeDestination("w2", Destination{ID: "d2", URL: "https://b.com"}, ann)), ErrLimit)
}

func TestCheckAddress(t *testing.T) {
	f := setup(t)
	f.destination("d1", "https://hooks.example.com", "10.1.0.0/16", "fd12::/16")
	cases := map[string]string{
		"8.8.8.8": "", "2001:4860:4860::8888": "", "3000::1": "", "3fff:ffff::1": "", "10.1.2.3": "", "fd12::1": "", "::ffff:8.8.8.8": "",
		"169.254.169.254": ReasonMetadataAddress, "169.254.1.1": ReasonMetadataAddress, "fe80::1": ReasonMetadataAddress, "192.0.0.192": ReasonMetadataAddress,
		"100.100.100.200": ReasonMetadataAddress, "168.63.129.16": ReasonMetadataAddress, "fd00:ec2::254": ReasonMetadataAddress, "fd20:ce::254": ReasonMetadataAddress, "::ffff:169.254.169.254": ReasonMetadataAddress,
		"127.0.0.1": ReasonForbiddenAddress, "::1": ReasonForbiddenAddress, "::": ReasonForbiddenAddress, "0.0.0.0": ReasonForbiddenAddress, "255.255.255.255": ReasonForbiddenAddress,
		"224.0.0.1": ReasonForbiddenAddress, "240.0.0.1": ReasonForbiddenAddress, "192.0.2.1": ReasonForbiddenAddress, "198.18.0.1": ReasonForbiddenAddress,
		"198.51.100.1": ReasonForbiddenAddress, "203.0.113.1": ReasonForbiddenAddress, "192.88.99.1": ReasonForbiddenAddress, "192.0.0.9": ReasonForbiddenAddress,
		"::7f00:1": ReasonForbiddenAddress, "::ffff:0:7f00:1": ReasonForbiddenAddress, "64:ff9b::a9fe:a9fe": ReasonForbiddenAddress, "64:ff9b:1::a9fe:a9fe": ReasonForbiddenAddress,
		"100::1": ReasonForbiddenAddress, "2001::1": ReasonForbiddenAddress, "2001:db8::1": ReasonForbiddenAddress, "3fff::1": ReasonForbiddenAddress, "2002:a9fe:a9fe::1": ReasonForbiddenAddress,
		"5f00::1": ReasonForbiddenAddress, "fec0::1": ReasonForbiddenAddress, "ff02::1": ReasonForbiddenAddress, "4000::1": ReasonForbiddenAddress, "1000::1": ReasonForbiddenAddress,
		"::ffff:127.0.0.1": ReasonForbiddenAddress, "10.2.0.1": ReasonPrivateAddress, "172.16.0.1": ReasonPrivateAddress, "192.168.1.1": ReasonPrivateAddress,
		"100.64.0.1": ReasonPrivateAddress, "fd13::1": ReasonPrivateAddress, "::ffff:192.168.1.1": ReasonPrivateAddress,
	}
	for text, want := range cases {
		err := f.r.CheckAddress("w1", "d1", netip.AddrPortFrom(netip.MustParseAddr(text), 443))
		if want == "" {
			if err != nil {
				t.Errorf("%s: %v", text, err)
			}
			continue
		}
		reason(t, text, err, want)
	}
	reason(t, "a zone", f.r.CheckAddress("w1", "d1", netip.AddrPortFrom(netip.MustParseAddr("fd12::1%eth0"), 443)), ReasonForbiddenAddress)
	if classify(netip.Addr{}, nil) != ReasonForbiddenAddress {
		t.Errorf("an invalid address is not forbidden")
	}
	reason(t, "another port", f.r.CheckAddress("w1", "d1", netip.MustParseAddrPort("8.8.8.8:80")), ReasonPortMismatch)
	expect(t, "no address", f.r.CheckAddress("w1", "d1", netip.AddrPort{}), ErrInvalid)
	expect(t, "no workspace", f.r.CheckAddress("", "d1", netip.MustParseAddrPort("8.8.8.8:443")), ErrInvalid)
	expect(t, "no destination", f.r.CheckAddress("w1", "", netip.MustParseAddrPort("8.8.8.8:443")), ErrInvalid)
	expect(t, "an unknown destination", f.r.CheckAddress("w1", "d9", netip.MustParseAddrPort("8.8.8.8:443")), ErrNotFound)
	for _, step := range []struct {
		address string
		want    string
	}{{"93.184.216.34", ""}, {"169.254.169.254", ReasonMetadataAddress}, {"93.184.216.34", ""}, {"127.0.0.1", ReasonForbiddenAddress}} {
		err := f.r.CheckAddress("w1", "d1", netip.AddrPortFrom(netip.MustParseAddr(step.address), 443))
		if step.want == "" && err != nil || step.want != "" && err == nil {
			t.Errorf("rebinding step %s: %v", step.address, err)
		}
	}
	if !allowable(netip.MustParsePrefix("10.0.0.0/8")) || !allowable(netip.MustParsePrefix("10.1.2.3/32")) || allowable(netip.MustParsePrefix("::ffff:10.0.0.0/104")) {
		t.Errorf("allowlist shapes")
	}
}

func TestCheckRedirect(t *testing.T) {
	f := setup(t)
	f.destination("d1", "https://a.example/hooks/")
	f.destination("d2", "https://b.example:8443")
	_, err := f.r.ProposeDestination("w1", Destination{ID: "d3", URL: "https://c.example"}, ann)
	f.must(err)
	f.destination("d4", "https://d.example")
	f.must(second(f.r.RevokeDestination("w1", "d4", ann)))
	base := "https://a.example:443/hooks/in"
	good := map[string]Redirect{
		"next":                       {Destination: "d1", URL: "https://a.example:443/hooks/next", Hop: 1},
		"/root?x=1#frag":             {Destination: "d1", URL: "https://a.example:443/root?x=1", Hop: 1},
		"https://B.EXAMPLE:8443/z":   {Destination: "d2", URL: "https://b.example:8443/z", Hop: 1},
		"//b.example:8443/y":         {Destination: "d2", URL: "https://b.example:8443/y", Hop: 1},
		"https://a.example./hooks/x": {Destination: "d1", URL: "https://a.example:443/hooks/x", Hop: 1},
	}
	for location, want := range good {
		if got, err := f.r.CheckRedirect("w1", "d1", base, 0, location); err != nil || got != want {
			t.Errorf("%s: %+v %v; want %+v", location, got, err, want)
		}
	}
	for _, location := range []string{
		"http://a.example/x", "https://a.example:444/x", "https://b.example/x", "https://c.example/x", "https://d.example/x", "https://user@a.example/x",
		"https://a.example\\@evil.example/", "\\\\evil.example/", "/a\\b", "\\x/", "/a b", "/é", "https://evil.example/", " https://a.example/", "https://a.example/\x7f", "%zz", "https://2130706433/",
	} {
		reason(t, location, second(f.r.CheckRedirect("w1", "d1", base, 0, location)), ReasonUnapprovedRedirect)
	}
	long := "/" + strings.Repeat("a", maxLocation-1)
	if got, err := f.r.CheckRedirect("w1", "d1", base, 0, long); err != nil || got.URL != "https://a.example:443"+long {
		t.Errorf("a location of the maximum length %+v %v", got, err)
	}
	g := setup(t)
	g.destination("d1", "https://a.example")
	_, err = g.r.ProposeDestination("w2", Destination{ID: "d1", URL: "https://x.example"}, ann)
	g.must(err)
	g.must(second(g.r.ApproveDestination("w2", "d1", bob)))
	reason(t, "another workspace's destination", second(g.r.CheckRedirect("w1", "d1", "https://a.example:443/", 0, "https://x.example/")), ReasonUnapprovedRedirect)
	if got, err := f.r.CheckRedirect("w1", "d2", "https://b.example:8443/z/w", 1, "v"); err != nil || got.URL != "https://b.example:8443/z/v" || got.Hop != 2 {
		t.Errorf("a second hop %+v %v", got, err)
	}
	reason(t, "the redirect limit", second(f.r.CheckRedirect("w1", "d2", "https://b.example:8443/z", limits.Redirects, "v")), ReasonRedirectLimit)
	expect(t, "a current url of another destination", second(f.r.CheckRedirect("w1", "d1", "https://b.example:8443/z", 0, "v")), ErrInvalid)
	expect(t, "an invalid current url", second(f.r.CheckRedirect("w1", "d1", "http://a.example/", 0, "v")), ErrInvalid)
	expect(t, "a negative hop", second(f.r.CheckRedirect("w1", "d1", base, -1, "v")), ErrInvalid)
	expect(t, "a long location", second(f.r.CheckRedirect("w1", "d1", base, 0, strings.Repeat("a", maxLocation+1))), ErrInvalid)
	expect(t, "no workspace", second(f.r.CheckRedirect("", "d1", base, 0, "v")), ErrInvalid)
	expect(t, "no destination", second(f.r.CheckRedirect("w1", "", base, 0, "v")), ErrInvalid)
	expect(t, "an unknown destination", second(f.r.CheckRedirect("w1", "d9", base, 0, "v")), ErrNotFound)
	reason(t, "from a proposed destination", second(f.r.CheckRedirect("w1", "d3", "https://c.example:443/", 0, "v")), ReasonNotActive)
	redirect, err := f.r.CheckRedirect("w1", "d1", base, 0, "https://b.example:8443/z")
	f.must(err)
	f.must(second(f.r.RevokeDestination("w1", redirect.Destination, ann)))
	reason(t, "revoked between the redirect and the dial", f.r.CheckAddress("w1", redirect.Destination, netip.MustParseAddrPort("8.8.8.8:8443")), ReasonNotActive)
}
