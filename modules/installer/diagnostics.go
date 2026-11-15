package installer

import (
	"net/netip"
)

var guidance = map[string]Guidance{
	"tls/unknown_authority": {Cause: "untrusted_issuer", Steps: []string{"add_internal_ca_to_product_trust", "verify_chain"}},
	"tls/hostname_mismatch": {Cause: "certificate_name", Steps: []string{"use_certificate_hostname", "reissue_certificate"}},
	"tls/expired":           {Cause: "validity_window", Steps: []string{"check_clock", "renew_certificate"}},
	"tls/not_yet_valid":     {Cause: "validity_window", Steps: []string{"check_clock", "renew_certificate"}},
	"tls/weak_signature":    {Cause: "weak_algorithm", Steps: []string{"reissue_certificate"}},
	"proxy/proxy_auth":      {Cause: "proxy_credentials", Steps: []string{"configure_proxy_credentials"}},
	"proxy/proxy_refused":   {Cause: "proxy_policy", Steps: []string{"configure_proxy_allowlist"}},
	"dns/dns_failure":       {Cause: "name_resolution", Steps: []string{"check_dns"}},
	"tcp/refused":           {Cause: "network_path", Steps: []string{"check_firewall", "check_service"}},
	"tcp/timeout":           {Cause: "network_path", Steps: []string{"check_firewall", "check_service"}},
}

func diagnosis(kind, failure string) (Guidance, bool) {
	switch kind {
	case DiagnoseDNS, DiagnoseTCP, DiagnoseProxy, DiagnoseTLS:
	default:
		return Guidance{}, false
	}
	switch failure {
	case "ok":
		return Guidance{Cause: "none", Steps: []string{}}, true
	case "unknown":
		return Guidance{Cause: "unclassified", Steps: []string{"collect_details"}}, true
	}
	g, ok := guidance[kind+"/"+failure]
	g.Steps = append([]string(nil), g.Steps...)
	return g, ok
}

func (r *Registry) CheckTarget(id, destination string, actor Actor) (Destination, error) {
	r.enter()
	defer r.leave()
	if !validID(destination) {
		return Destination{}, ErrInvalid
	}
	s, err := r.probe(id, actor)
	if err != nil {
		return Destination{}, err
	}
	d, ok := s.destinations[destination]
	if !ok {
		return Destination{}, refuse(ReasonNotApproved)
	}
	d.Ranges = append([]string(nil), d.Ranges...)
	return d, nil
}

func (r *Registry) Report(id, destination string, result Result, actor Actor) (Guidance, error) {
	r.enter()
	defer r.leave()
	g, known := diagnosis(result.Kind, result.Failure)
	resolved := !(result.Kind == DiagnoseDNS && result.Failure != "ok")
	addr, err := netip.ParseAddr(result.Address)
	switch {
	case !known || !validID(destination) || result.Via != "" && !validID(result.Via) || result.Kind == DiagnoseProxy && result.Via == "":
		return Guidance{}, ErrInvalid
	case resolved && (err != nil || addr.Zone() != "" || addr.Is4In6() || addr.String() != result.Address):
		return Guidance{}, ErrInvalid
	case !resolved && result.Address != "":
		return Guidance{}, ErrInvalid
	}
	s, err := r.probe(id, actor)
	if err != nil {
		return Guidance{}, err
	}
	d, ok := s.destinations[destination]
	proxy, via := s.destinations[result.Via]
	if result.Kind == DiagnoseProxy {
		d = proxy
	}
	switch {
	case !ok || result.Via != "" && (result.Via == destination || !via):
		return Guidance{}, refuse(ReasonNotApproved)
	case resolved && !within(d.Ranges, addr):
		return Guidance{}, refuse(ReasonAddressNotApproved)
	}
	s.reports = append(s.reports, result)
	if len(s.reports) > r.limits.Reports {
		s.reports = s.reports[1:]
	}
	return g, nil
}

func within(ranges []string, addr netip.Addr) bool {
	for _, text := range ranges {
		if p, _ := validRange(text); p.Contains(addr) {
			return true
		}
	}
	return false
}

func (r *Registry) Reports(id string) ([]Result, error) {
	r.enter()
	defer r.leave()
	s, err := r.site(id)
	if err != nil {
		return nil, err
	}
	return append([]Result{}, s.reports...), nil
}
