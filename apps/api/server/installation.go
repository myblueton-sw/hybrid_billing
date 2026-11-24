package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/installer"
)

const (
	installCommand   = "installation."
	installTarget    = "installation"
	proposalTarget   = "installation-proposal-"
	installApprove   = "approve"
	installWithdraw  = "withdraw"
	installPlan      = "propose_plan"
	ownProposal      = "own_proposal"
	notObserved      = "not_observed"
	previewAvailable = "available"
	previewNone      = "unavailable"
	notApplicable    = "not_applicable"
	notInstalled     = "not_installed"
)

type installItemRow struct {
	Kind     string `json:"kind"`
	Target   string `json:"target"`
	Action   string `json:"action"`
	Conflict string `json:"conflict"`
}

type installEffectRow struct {
	Volume string `json:"volume"`
	Effect string `json:"effect"`
}

type installPreviewRow struct {
	State       string             `json:"state"`
	Reason      string             `json:"reason"`
	Kind        string             `json:"kind"`
	Version     string             `json:"version"`
	Hash        string             `json:"hash"`
	Services    []installItemRow   `json:"services"`
	Ports       []installItemRow   `json:"ports"`
	Volumes     []installItemRow   `json:"volumes"`
	Data        []installEffectRow `json:"data_effects"`
	Permissions []string           `json:"permissions"`
	Leftovers   []installItemRow   `json:"leftovers"`
	Conflicts   []installItemRow   `json:"conflicts"`
}

type installOverrideRow struct {
	Kind   string `json:"kind"`
	Target string `json:"target"`
	Value  string `json:"value"`
}

type installSiteRow struct {
	ID        string               `json:"id"`
	Platform  string               `json:"platform"`
	Probe     string               `json:"probe"`
	Overrides []installOverrideRow `json:"config_overrides"`
}

type installObservationRow struct {
	State      string `json:"state"`
	ObservedAt string `json:"observed_at"`
	ExpiresAt  string `json:"expires_at"`
}

type installOwnedServiceRow struct {
	Name  string   `json:"name"`
	Ports []string `json:"ports"`
}

type installOwnedVolumeRow struct {
	Host     string `json:"host"`
	Logical  string `json:"logical"`
	Data     bool   `json:"data"`
	Detached bool   `json:"detached"`
}

type installHistoryRow struct {
	Kind   string   `json:"kind"`
	From   string   `json:"from"`
	To     string   `json:"to"`
	Run    string   `json:"run"`
	Backup string   `json:"backup"`
	Runs   []string `json:"runs"`
}

type installStateRow struct {
	State    string                   `json:"state"`
	Version  string                   `json:"version"`
	Services []installOwnedServiceRow `json:"services"`
	Volumes  []installOwnedVolumeRow  `json:"volumes"`
	History  []installHistoryRow      `json:"history"`
	Backups  []string                 `json:"backups"`
}

type installServiceRow struct {
	Name     string   `json:"name"`
	Artifact string   `json:"artifact"`
	Ports    []string `json:"ports"`
	Volumes  []string `json:"volumes"`
}

type installVolumeRow struct {
	Name string `json:"name"`
	Data bool   `json:"data"`
}

type installArtifactRow struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type installMarkRow struct {
	By               string `json:"by"`
	ByLogin          string `json:"by_login"`
	Reason           string `json:"reason"`
	ConfirmedBy      string `json:"confirmed_by"`
	ConfirmedByLogin string `json:"confirmed_by_login"`
}

type installReleaseRow struct {
	Version     string               `json:"version"`
	Key         string               `json:"key_version"`
	UpgradeFrom []string             `json:"upgrade_from"`
	Platforms   []string             `json:"platforms"`
	Services    []installServiceRow  `json:"services"`
	Volumes     []installVolumeRow   `json:"volumes"`
	Artifacts   []installArtifactRow `json:"artifacts"`
	Mark        *installMarkRow      `json:"mark"`
}

type installSummaryRow struct {
	Key         string   `json:"key_version"`
	Release     string   `json:"release_version"`
	Site        string   `json:"site"`
	Destination string   `json:"destination"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Ranges      []string `json:"ranges"`
	Version     string   `json:"plan_version"`
	Hash        string   `json:"plan_hash"`
	Admin       string   `json:"plan_admin"`
}

type installProposalRow struct {
	Number           uint64            `json:"number"`
	Target           string            `json:"target"`
	Kind             string            `json:"kind"`
	State            string            `json:"state"`
	Created          string            `json:"created_at"`
	Proposer         string            `json:"proposer"`
	ProposerLogin    string            `json:"proposer_login"`
	Approver         string            `json:"approver"`
	ApproverLogin    string            `json:"approver_login"`
	Rationale        string            `json:"rationale"`
	ClosedBy         string            `json:"closed_by"`
	ClosedByLogin    string            `json:"closed_by_login"`
	ClosingRationale string            `json:"closing_rationale"`
	Summary          installSummaryRow `json:"summary"`
	CanApprove       bool              `json:"can_approve"`
	ApproveBlocked   string            `json:"approve_blocked"`
	CanWithdraw      bool              `json:"can_withdraw"`
	WithdrawBlocked  string            `json:"withdraw_blocked"`
	CurrentHash      string            `json:"current_hash"`
}

type installDestinationRow struct {
	ID     string   `json:"id"`
	Host   string   `json:"host"`
	Port   int      `json:"port"`
	Ranges []string `json:"ranges"`
}

type installReportRow struct {
	Destination string   `json:"destination"`
	Kind        string   `json:"kind"`
	Failure     string   `json:"failure"`
	Address     string   `json:"address"`
	Via         string   `json:"via"`
	Cause       string   `json:"cause"`
	Steps       []string `json:"steps"`
}

type installationEmpty struct {
	Revision string `json:"revision"`
	Now      string `json:"now"`
	State    string `json:"state"`
}

type installationPage struct {
	Revision     string                  `json:"revision"`
	Now          string                  `json:"now"`
	State        string                  `json:"state"`
	Site         installSiteRow          `json:"site"`
	Observation  installObservationRow   `json:"observation"`
	Installed    installStateRow         `json:"installed"`
	Releases     []installReleaseRow     `json:"releases"`
	Preview      installPreviewRow       `json:"preview"`
	PreviewBasis string                  `json:"preview_basis"`
	Upgrade      installPreviewRow       `json:"upgrade"`
	Rollback     installPreviewRow       `json:"rollback"`
	Proposals    []installProposalRow    `json:"proposals"`
	Destinations []installDestinationRow `json:"destinations"`
	Reports      []installReportRow      `json:"reports"`
	Runs         string                  `json:"runs"`
}

func items(list []installer.Item) []installItemRow {
	out := []installItemRow{}
	for _, i := range list {
		out = append(out, installItemRow{Kind: i.Kind, Target: i.Target, Action: i.Action, Conflict: i.Conflict})
	}
	return out
}

func installReason(err error) string {
	var refusal installer.Refusal
	if errors.As(err, &refusal) {
		return refusal.Reason
	}
	return notObserved
}

func previewOf(kind, version string, v installer.PreviewView, err error) installPreviewRow {
	row := installPreviewRow{State: previewAvailable, Kind: kind, Version: version, Hash: v.Hash, Services: items(v.Services), Ports: items(v.Ports), Volumes: items(v.Volumes),
		Data: []installEffectRow{}, Permissions: append([]string{}, v.Permissions...), Leftovers: items(v.Leftovers), Conflicts: items(v.Conflicts)}
	for _, d := range v.Data {
		row.Data = append(row.Data, installEffectRow{Volume: d.Volume, Effect: d.Effect})
	}
	if err != nil {
		row.State, row.Reason = previewNone, installReason(err)
	}
	return row
}

func absentPreview(kind string) installPreviewRow {
	row := previewOf(kind, "", installer.PreviewView{}, nil)
	row.State = notApplicable
	return row
}

func observationState(errs []error) string {
	for _, err := range errs {
		switch {
		case err == nil:
			return "fresh"
		case installReason(err) == notObserved:
			return "missing"
		case installReason(err) == installer.ReasonObservationStale:
			return "stale"
		}
	}
	return "unknown"
}

func installedOf(v installer.InstalledView) installStateRow {
	row := installStateRow{State: v.State, Version: v.Version, Services: []installOwnedServiceRow{}, Volumes: []installOwnedVolumeRow{}, History: []installHistoryRow{},
		Backups: append([]string{}, v.Backups...)}
	for _, svc := range v.Services {
		row.Services = append(row.Services, installOwnedServiceRow{Name: svc.Name, Ports: append([]string{}, svc.Ports...)})
	}
	for _, vol := range v.Volumes {
		row.Volumes = append(row.Volumes, installOwnedVolumeRow{Host: vol.Host, Logical: vol.Logical, Data: vol.Data, Detached: vol.Detached})
	}
	for _, h := range v.History {
		row.History = append(row.History, installHistoryRow{Kind: h.Kind, From: h.From, To: h.To, Run: h.Run, Backup: h.Backup, Runs: append([]string{}, h.Runs...)})
	}
	return row
}

func (s *Server) releaseOf(rel installer.Release, mark *installer.Mark) installReleaseRow {
	row := installReleaseRow{Version: rel.Version, Key: rel.Key, UpgradeFrom: append([]string{}, rel.UpgradeFrom...), Platforms: []string{}, Services: []installServiceRow{},
		Volumes: []installVolumeRow{}, Artifacts: []installArtifactRow{}}
	for _, p := range rel.Platforms {
		row.Platforms = append(row.Platforms, p.OS+"/"+p.Arch)
	}
	for _, svc := range rel.Services {
		ports := []string{}
		for _, p := range svc.Ports {
			ports = append(ports, strconv.Itoa(p.Number)+"/"+p.Protocol)
		}
		row.Services = append(row.Services, installServiceRow{Name: svc.Name, Artifact: svc.Artifact, Ports: ports, Volumes: append([]string{}, svc.Volumes...)})
	}
	for _, v := range rel.Volumes {
		row.Volumes = append(row.Volumes, installVolumeRow{Name: v.Name, Data: v.Data})
	}
	for _, a := range rel.Artifacts {
		row.Artifacts = append(row.Artifacts, installArtifactRow{Name: a.Name, Kind: a.Kind})
	}
	if mark != nil {
		row.Mark = &installMarkRow{By: mark.By, ByLogin: s.loginOf(mark.By), Reason: mark.Reason, ConfirmedBy: mark.ConfirmedBy, ConfirmedByLogin: s.loginOf(mark.ConfirmedBy)}
	}
	return row
}

func firstOf(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func summaryOf(c installer.Change) installSummaryRow {
	return installSummaryRow{Key: c.Key.Version, Release: c.Release.Version, Site: firstOf(c.Site.ID, c.Destination.Site, c.Acceptance.Site), Destination: c.Destination.ID,
		Host: c.Destination.Host, Port: c.Destination.Port, Ranges: append([]string{}, c.Destination.Ranges...), Version: c.Acceptance.Version, Hash: c.Acceptance.Hash,
		Admin: c.Acceptance.Admin}
}

func (s *Server) blockers(v installer.ProposalView, reader string) (string, string, string) {
	if v.State != installer.ProposalPending {
		return installer.ReasonNotPending, installer.ReasonNotPending, ""
	}
	withdraw := ""
	if v.Proposer != reader {
		withdraw = installer.ReasonNotProposer
	}
	switch {
	case v.Proposer == reader:
		return ownProposal, withdraw, ""
	case v.Change.Kind != installer.ChangePlan:
		return "", withdraw, ""
	}
	a := v.Change.Acceptance
	current, err := s.installer.Preview(a.Site, a.Version)
	switch {
	case err != nil:
		return installReason(err), withdraw, ""
	case current.Hash != a.Hash:
		return installer.ReasonPlanChanged, withdraw, current.Hash
	case len(current.Conflicts) > 0:
		return installer.ReasonConflicts, withdraw, ""
	}
	return "", withdraw, ""
}

func (s *Server) installProposalRow(e *installEntry, reader string) installProposalRow {
	v := e.view
	row := installProposalRow{Number: v.Number, Target: proposalTarget + strconv.FormatUint(v.Number, 10), Kind: v.Change.Kind, State: v.State, Created: stamp(e.created),
		Proposer: v.Proposer, ProposerLogin: s.loginOf(v.Proposer), Approver: v.Approver, ApproverLogin: s.loginOf(v.Approver), Rationale: e.rationale,
		ClosedBy: e.closedBy, ClosedByLogin: s.loginOf(e.closedBy), ClosingRationale: e.closing, Summary: summaryOf(v.Change)}
	row.ApproveBlocked, row.WithdrawBlocked, row.CurrentHash = s.blockers(v, reader)
	row.CanApprove, row.CanWithdraw = row.ApproveBlocked == "", row.WithdrawBlocked == ""
	return row
}

func (s *Server) planTarget(installed string) (string, string) {
	for index := len(s.install.entries) - 1; index >= 0; index-- {
		if v := s.install.entries[index].view; v.State == installer.ProposalPending && v.Change.Kind == installer.ChangePlan {
			return v.Change.Acceptance.Version, "pending_plan"
		}
	}
	if version := s.newestOther(installed); version != "" {
		return version, "newest_release"
	}
	return "", ""
}

func (s *Server) newestOther(installed string) string {
	for index := len(s.install.releases) - 1; index >= 0; index-- {
		if version := s.install.releases[index]; version != installed {
			return version
		}
	}
	return ""
}

func (s *Server) installationPage(reader string, now time.Time) (installationPage, error) {
	if s.faulty("installer-read") {
		return installationPage{}, installer.ErrInvalid
	}
	id := s.install.site
	page := installationPage{Revision: s.Revision(), Now: stamp(now), State: "configured", Releases: []installReleaseRow{}, Proposals: []installProposalRow{},
		Destinations: []installDestinationRow{}, Reports: []installReportRow{}, Runs: "none", Upgrade: absentPreview(installer.RunUpgrade), Rollback: absentPreview(installer.RunRollback),
		Preview: absentPreview(installer.RunInstall), Installed: installStateRow{State: notInstalled, Services: []installOwnedServiceRow{}, Volumes: []installOwnedVolumeRow{},
			History: []installHistoryRow{}, Backups: []string{}}}
	site, err := s.installer.Site(id)
	errs := []error{err}
	installed, missing := s.installer.Installed(id)
	if !errors.Is(missing, installer.ErrNotFound) {
		errs = append(errs, missing)
	}
	for index := len(s.install.releases) - 1; index >= 0; index-- {
		rel, mark, err := s.installer.Release(s.install.releases[index])
		errs = append(errs, err)
		page.Releases = append(page.Releases, s.releaseOf(rel, mark))
	}
	if err := errors.Join(errs...); err != nil {
		return installationPage{}, err
	}
	page.Site = installSiteRow{ID: site.ID, Platform: site.Platform.OS + "/" + site.Platform.Arch, Probe: site.Probe, Overrides: []installOverrideRow{}}
	probes := []error{}
	if target, basis := s.planTarget(installed.Version); target != "" {
		page.PreviewBasis = basis
		v, err := s.installer.Preview(id, target)
		page.Preview, probes = previewOf(installer.RunInstall, target, v, err), append(probes, err)
	}
	if missing == nil {
		page.Installed = installedOf(installed)
		if next := s.newestOther(installed.Version); next != "" {
			v, err := s.installer.PreviewUpgrade(id, next)
			page.Upgrade, probes = previewOf(installer.RunUpgrade, next, v, err), append(probes, err)
		}
		v, err := s.installer.PreviewRollback(id)
		page.Rollback, probes = previewOf(installer.RunRollback, v.Version, v, err), append(probes, err)
	}
	page.Observation = installObservationRow{State: observationState(probes), ObservedAt: stamp(s.install.observed),
		ExpiresAt: stamp(s.install.observed.Add(InstallerLimits().ObservationLife))}
	for index := len(s.install.entries) - 1; index >= 0 && len(page.Proposals) < installShown; index-- {
		page.Proposals = append(page.Proposals, s.installProposalRow(s.install.entries[index], reader))
	}
	for _, d := range s.install.destinations {
		page.Destinations = append(page.Destinations, installDestinationRow{ID: d.ID, Host: d.Host, Port: d.Port, Ranges: append([]string{}, d.Ranges...)})
	}
	for _, r := range s.install.reports {
		page.Reports = append(page.Reports, installReportRow{Destination: r.destination, Kind: r.result.Kind, Failure: r.result.Failure, Address: r.result.Address,
			Via: r.result.Via, Cause: r.guidance.Cause, Steps: append([]string{}, r.guidance.Steps...)})
	}
	return page, nil
}

func (s *Server) readInstallation(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	_, answer, ok := s.admitted(w, r)
	if !ok {
		return
	}
	now := s.clock()
	if s.install.site == "" || answer.Workspace() != s.bootWorkspace {
		writeJSON(w, http.StatusOK, request, installationEmpty{Revision: s.Revision(), Now: stamp(now), State: "not_configured"})
		return
	}
	page, err := s.installationPage(answer.User(), now)
	if err != nil {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	writeJSON(w, http.StatusOK, request, page)
}

func (s *Server) installEntry(target string) *installEntry {
	raw, found := strings.CutPrefix(target, proposalTarget)
	number, err := strconv.ParseUint(raw, 10, 64)
	if !found || err != nil || strconv.FormatUint(number, 10) != raw {
		return nil
	}
	return s.install.entry(number)
}

func mapInstaller(err error) error {
	var refusal installer.Refusal
	switch {
	case err == nil:
		return nil
	case errors.As(err, &refusal):
		if refusal.Reason == installer.ReasonSelfApproval || refusal.Reason == installer.ReasonNotProposer {
			return refuse(commands.PrerequisiteFailed)
		}
		return refuse(commands.InvalidStage)
	case errors.Is(err, installer.ErrNotFound), errors.Is(err, installer.ErrConflict):
		return refuse(commands.InvalidStage)
	case errors.Is(err, installer.ErrLimit):
		return refuse(commands.PrerequisiteFailed)
	}
	return refuse(commands.InternalError)
}

type installHandler struct {
	s    *Server
	kind string
}

func (h installHandler) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	s := h.s
	found := workspace == s.bootWorkspace && s.install.site != ""
	if h.kind == installPlan {
		found = found && target == installTarget
	} else {
		found = found && s.installEntry(target) != nil
	}
	if !found {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: s.Revision()}, commands.Found
}

func validPlanHash(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	for _, c := range hash {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (s *Server) proposePlan(ctx commands.Context, reader installer.Actor) (*installEntry, error) {
	version, hash := ctx.Fields["version"], ctx.Fields["hash"]
	known := false
	for _, v := range s.install.releases {
		known = known || v == version
	}
	switch {
	case !known:
		return nil, refuse(commands.InvalidRequest, "version")
	case !validPlanHash(hash):
		return nil, refuse(commands.InvalidRequest, "hash")
	}
	e, err := s.installPropose(installer.Change{Kind: installer.ChangePlan, Acceptance: installer.Acceptance{Site: s.install.site, Version: version, Hash: hash,
		Admin: reader.ID}}, reader, ctx.Rationale)
	return e, mapInstaller(err)
}

func (h installHandler) Apply(ctx commands.Context) (commands.Outcome, error) {
	s := h.s
	if len(ctx.Rationale) > maxRationale {
		return commands.Outcome{}, refuse(commands.InvalidRequest)
	}
	reader := installer.Actor{ID: ctx.User, Kind: installer.KindHuman}
	e := s.installEntry(ctx.Target)
	var err error
	switch h.kind {
	case installApprove:
		err = mapInstaller(s.installApprove(e, reader, ctx.Rationale))
	case installWithdraw:
		err = mapInstaller(s.installWithdraw(e, reader, ctx.Rationale))
	default:
		e, err = s.proposePlan(ctx, reader)
	}
	if err != nil {
		return commands.Outcome{}, err
	}
	number := strconv.FormatUint(e.view.Number, 10)
	return commands.Outcome{ResultRevision: s.Next(), BusinessState: e.view.State, Fields: map[string]string{"proposal": number, "target": proposalTarget + number}}, nil
}

func (s *Server) installationDefinitions() []commands.Definition {
	permission := commands.Permission{Action: gate.AdministerInstallation, Kind: gate.InstallationKind}
	define := func(kind string, f ...commands.Field) commands.Definition {
		return commands.Definition{ID: installCommand + kind, Permission: permission, Mutating: true, Rationale: true, Fields: f, Handler: installHandler{s: s, kind: kind}}
	}
	return []commands.Definition{
		define(installApprove),
		define(installWithdraw),
		define(installPlan, commands.Field{Name: "version", Type: commands.Text, Required: true}, commands.Field{Name: "hash", Type: commands.Text, Required: true}),
	}
}
