package thirdparty

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxSourceBytes = 200

var (
	unpinnedLabels = []string{"latest", "main", "master", "head", "stable", "nightly", "dev", "trunk"}
	licenses       = map[string][3]string{
		"MIT": {"all", "all", "all"}, "Apache-2.0": {"all", "all", "all"}, "BSD-2-Clause": {"all", "all", "all"}, "BSD-3-Clause": {"all", "all", "all"},
		"ISC": {"all", "all", "all"}, "Zlib": {"all", "all", "all"}, "OFL-1.1": {"all", "all", "all"}, "CC-BY-4.0": {"all", "all", "all"},
		"MPL-2.0": {"unmodified", "all", "all"}, "CC-BY-SA-4.0": {"none", "all", "all"}, "LGPL-2.1-only": {"none", "all", "all"},
		"LGPL-3.0-only": {"none", "all", "all"}, "GPL-2.0-only": {"none", "none", "all"}, "GPL-3.0-only": {"none", "none", "all"},
		"AGPL-3.0-only": {"none", "none", "all"}, "PolyForm-Noncommercial-1.0.0": {"none", "none", "none"},
	}
)

func oneOf(value string, set ...string) bool {
	for _, item := range set {
		if item == value {
			return true
		}
	}
	return false
}

func hexOf(text string, sizes ...int) bool {
	if !oneOf(strconv.Itoa(len(text)), func() []string {
		out := []string{}
		for _, size := range sizes {
			out = append(out, strconv.Itoa(size))
		}
		return out
	}()...) {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func digestOK(text string) bool { return hexOf(text, 64) }

func validSource(text string) bool {
	if len(text) > maxSourceBytes || !strings.HasPrefix(text, "https://") {
		return false
	}
	for index := 0; index < len(text); index++ {
		if text[index] <= 0x20 || text[index] >= 0x7f {
			return false
		}
	}
	u, err := url.Parse(text)
	return err == nil && u.Host != "" && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && !strings.Contains(text, "#")
}

func pinned(version string) bool {
	if oneOf(strings.ToLower(version), unpinnedLabels...) {
		return false
	}
	for _, segment := range strings.Split(version, ".") {
		if segment == "x" || segment == "X" {
			return false
		}
	}
	return true
}

func allowed(license, distribution, modification string) bool {
	if distribution == DistNotShipped {
		return true
	}
	rule := licenses[license][map[string]int{DistBundled: 0, DistSeparate: 1, DistToolOnly: 2}[distribution]]
	return rule == "all" || rule == "unmodified" && modification == ModUnmodified
}

func (c Component) valid() bool {
	if !validID(c.ID) || !validID(c.Name) || !validID(c.Version) || !hexOf(c.Commit, 40, 64) || !validSource(c.Source) || !validID(c.License) ||
		!digestOK(c.LicenseText) || !digestOK(c.Notice) || !digestOK(c.SBOM) || !digestOK(c.Tests) || len(c.Dependencies) > maxDependencies ||
		!oneOf(c.Telemetry, TelemetryNone, TelemetryOptional, TelemetryRequired) || !oneOf(c.Modification, ModUnmodified, ModPatched, ModForked) ||
		!oneOf(c.Distribution, DistBundled, DistSeparate, DistToolOnly, DistNotShipped) {
		return false
	}
	if c.Supersedes != "" && (!validID(c.Supersedes) || c.Supersedes == c.ID) {
		return false
	}
	for index, id := range c.Dependencies {
		if !validID(id) || id == c.ID || index > 0 && c.Dependencies[index-1] >= id {
			return false
		}
	}
	shipped := c.Distribution == DistBundled || c.Distribution == DistSeparate
	return shipped == (c.Artifact != "") && (c.Artifact == "" || digestOK(c.Artifact)) &&
		(c.Modification == ModUnmodified) == (c.Patch == "") && (c.Patch == "" || digestOK(c.Patch)) &&
		(c.Telemetry == TelemetryOptional) == (c.TelemetryEvidence != "") && (c.TelemetryEvidence == "" || digestOK(c.TelemetryEvidence))
}

func clean(c Change) (Change, error) {
	out := Change{Kind: c.Kind}
	switch c.Kind {
	case ChangeComponent:
		out.Component = c.Component
		out = copyChange(out)
		switch {
		case c.Decision != (Decision{}) || !out.Component.valid():
			return out, ErrInvalid
		case !pinned(out.Component.Version):
			return out, refuse(ReasonUnpinnedVersion)
		case licenses[out.Component.License] == [3]string{}:
			return out, refuse(ReasonUnknownLicense)
		}
		return out, nil
	case ChangeAdopt:
		out.Decision = Decision{Component: c.Decision.Component}
		if validID(c.Decision.Component) && c.Decision.Reason == "" {
			return out, nil
		}
	case ChangeReject, ChangeRevoke:
		out.Decision = c.Decision
		if validID(c.Decision.Component) && validText(c.Decision.Reason) {
			return out, nil
		}
	}
	return out, ErrInvalid
}

func copyChange(c Change) Change {
	c.Component.Dependencies = append([]string(nil), c.Component.Dependencies...)
	return c
}

func copyProposal(p ProposalView) ProposalView {
	p.Change = copyChange(p.Change)
	return p
}

func boolText(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func encodeChange(c Change) []byte {
	var e encoder
	x := c.Component
	for _, value := range []string{c.Kind, x.ID, x.Name, x.Version, x.Commit, x.Source, x.License, x.LicenseText, x.Notice, x.SBOM, x.Artifact, x.Patch,
		boolText(x.Offline), x.Telemetry, x.TelemetryEvidence, x.Modification, x.Distribution, x.Tests, boolText(x.Passed), x.Supersedes, c.Decision.Component, c.Decision.Reason,
		strconv.Itoa(len(x.Dependencies))} {
		e.text(value)
	}
	for _, id := range x.Dependencies {
		e.text(id)
	}
	return e.out.Bytes()
}

func (r *Registry) check(c Change, _, _ string) error {
	if c.Kind == ChangeComponent {
		x := c.Component
		name := strings.ToLower(x.Name)
		for _, id := range x.Dependencies {
			if r.components[id] == nil {
				return ErrNotFound
			}
		}
		if r.components[x.ID] != nil || !r.free(append(append([]string(nil), r.versions[[2]string{name, x.Version}]...), r.commits[[2]string{name, x.Commit}]...), x.Supersedes) {
			return ErrConflict
		}
		shipped := x.Distribution == DistBundled || x.Distribution == DistSeparate
		for _, id := range x.Dependencies {
			if d := r.components[id].view.Distribution; shipped && (d == DistToolOnly || d == DistNotShipped) {
				return refuse(ReasonToolDependency)
			}
		}
		if len(r.components) >= maxComponents {
			return ErrLimit
		}
		return nil
	}
	k := r.components[c.Decision.Component]
	if k == nil {
		return ErrNotFound
	}
	switch c.Kind {
	case ChangeRevoke:
		if k.state != StateAdopted {
			return refuse(ReasonNotAdopted)
		}
		return nil
	case ChangeReject:
		if k.state != StateCandidate {
			return refuse(ReasonNotCandidate)
		}
		return nil
	}
	x := k.view
	switch {
	case k.state != StateCandidate:
		return refuse(ReasonNotCandidate)
	case !x.Passed:
		return refuse(ReasonTestsFailed)
	case !x.Offline && (x.Distribution == DistBundled || x.Distribution == DistSeparate):
		return refuse(ReasonNotOffline)
	case x.Telemetry == TelemetryRequired:
		return refuse(ReasonTelemetryRequired)
	case !allowed(x.License, x.Distribution, x.Modification):
		return refuse(ReasonLicenseBoundary)
	}
	for _, id := range x.Dependencies {
		d := r.components[id]
		switch {
		case d.state != StateAdopted:
			return refuse(ReasonDependencyNotAdopted)
		case !allowed(d.view.License, shipsIn(x, d.view), d.view.Modification):
			return refuse(ReasonDependencyBoundary)
		}
	}
	return nil
}

func (r *Registry) free(earlier []string, supersedes string) bool {
	latest := ""
	for _, id := range earlier {
		k := r.components[id]
		if k.state != StateRejected && k.state != StateRevoked {
			return false
		}
		if latest == "" || k.seq > r.components[latest].seq {
			latest = id
		}
	}
	return supersedes == latest
}

func shipsIn(parent, dependency Component) string {
	if dependency.Distribution == DistSeparate {
		return DistSeparate
	}
	return parent.Distribution
}

func (r *Registry) apply(c Change, _ people, _ time.Time) {
	c = copyChange(c)
	switch c.Kind {
	case ChangeComponent:
		x := c.Component
		name := strings.ToLower(x.Name)
		r.components[x.ID] = &component{view: x, state: StateCandidate, seq: len(r.components) + 1}
		r.versions[[2]string{name, x.Version}] = append(r.versions[[2]string{name, x.Version}], x.ID)
		r.commits[[2]string{name, x.Commit}] = append(r.commits[[2]string{name, x.Commit}], x.ID)
		return
	}
	k := r.components[c.Decision.Component]
	k.reason = c.Decision.Reason
	k.state = map[string]string{ChangeAdopt: StateAdopted, ChangeReject: StateRejected, ChangeRevoke: StateRevoked}[c.Kind]
}
