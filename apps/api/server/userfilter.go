package server

import (
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

const (
	userSearchBytes  = 64
	userPageDefault  = 50
	userPageMax      = 100
	segmentLogin     = "a:"
	segmentDeparted  = "d:"
	usersOrder       = "login_then_departed_user_id"
	membershipNone   = "none"
	membershipEnded  = "ended"
	signinPassword   = "password_set"
	signinSetLink    = "set_link"
	signinResetLink  = "reset_link"
	signinNotSet     = "not_set"
	invitationFilter = "pending"
	holderFilter     = "true"
)

var userFilterValues = map[string][]string{
	"status":     {string(identity.Active), string(identity.Suspended), string(identity.Departed)},
	"membership": {string(identity.MembershipActive), string(identity.MembershipSuspended), membershipEnded, membershipNone},
	"role":       {string(identity.Administrator), string(identity.Member)},
	"period":     {"open", "bounded", "ends_soon", "ended"},
	"signin":     {signinPassword, signinSetLink, signinResetLink, signinNotSet},
	"invitation": {invitationFilter},
	"holder":     {holderFilter},
}

type userFilter struct {
	Q          string `json:"q"`
	Status     string `json:"status"`
	Membership string `json:"membership"`
	Role       string `json:"role"`
	Period     string `json:"period"`
	Signin     string `json:"signin"`
	Invitation string `json:"invitation"`
	Holder     string `json:"holder"`
}

func (f *userFilter) field(key string) *string {
	return map[string]*string{"status": &f.Status, "membership": &f.Membership, "role": &f.Role, "period": &f.Period, "signin": &f.Signin,
		"invitation": &f.Invitation, "holder": &f.Holder}[key]
}

type userCursor struct {
	departed bool
	user     string
}

func (c userCursor) String() string {
	if c.departed {
		return segmentDeparted + c.user
	}
	return segmentLogin + c.user
}

type usersQuery struct {
	filter userFilter
	after  *userCursor
	limit  int
}

func parseCursor(raw string) (*userCursor, bool) {
	if user, ok := strings.CutPrefix(raw, segmentLogin); ok && validWorkspace(user) {
		return &userCursor{user: user}, true
	}
	if user, ok := strings.CutPrefix(raw, segmentDeparted); ok && validWorkspace(user) {
		return &userCursor{departed: true, user: user}, true
	}
	return nil, false
}

func parseSearch(raw string) (string, bool) {
	q := strings.TrimSpace(raw)
	return q, q != "" && len(q) <= userSearchBytes && utf8.ValidString(q)
}

func parseUsersQuery(values url.Values) (usersQuery, commands.Code, string) {
	out := usersQuery{limit: userPageDefault}
	method := false
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		list := values[key]
		if len(list) != 1 || list[0] == "" {
			return out, commands.InvalidRequest, key
		}
		raw, ok := list[0], true
		switch key {
		case "q":
			out.filter.Q, ok = parseSearch(raw)
		case "after":
			out.after, ok = parseCursor(raw)
		case "limit":
			n, err := strconv.Atoi(raw)
			out.limit, ok = n, err == nil && n >= 1 && n <= userPageMax
		case "method":
			method = true
		default:
			ok = slices.Contains(userFilterValues[key], raw)
			if ok {
				*out.filter.field(key) = raw
			}
		}
		if !ok {
			return out, commands.InvalidRequest, key
		}
	}
	if method {
		return out, commands.UnsupportedCapability, "method"
	}
	return out, "", ""
}

var filterKeys = []string{"holder", "invitation", "membership", "period", "q", "role", "signin", "status"}

func (f *userFilter) term(key string) *string {
	if key == "q" {
		return &f.Q
	}
	return f.field(key)
}

func (f userFilter) terms() []bulkrevoke.FilterTerm {
	out := []bulkrevoke.FilterTerm{}
	for _, key := range filterKeys {
		if value := *f.term(key); value != "" {
			out = append(out, bulkrevoke.FilterTerm{Key: key, Value: value})
		}
	}
	return out
}

func filterOf(terms []bulkrevoke.FilterTerm) userFilter {
	var f userFilter
	for _, term := range terms {
		if field := f.term(term.Key); field != nil {
			*field = term.Value
		}
	}
	return f
}

func foldRune(r rune) rune {
	least := r
	for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
		least = min(least, next)
	}
	return least
}

func fold(text string) string {
	var b strings.Builder
	for _, r := range text {
		b.WriteRune(foldRune(r))
	}
	return b.String()
}

func membershipValue(m *identity.MemberInfo, now time.Time) string {
	switch {
	case m == nil:
		return membershipNone
	case m.State == identity.MembershipSuspended:
		return string(identity.MembershipSuspended)
	case m.Until.IsZero() || now.Before(m.Until):
		return string(identity.MembershipActive)
	}
	return membershipEnded
}

func signinValue(row userRow) []string {
	out := []string{}
	if row.PasswordSet {
		out = append(out, signinPassword)
	}
	switch row.LinkPurpose {
	case passwords.PurposeSet:
		out = append(out, signinSetLink)
	case passwords.PurposeReset:
		out = append(out, signinResetLink)
	}
	if len(out) == 0 {
		out = append(out, signinNotSet)
	}
	return out
}

type userMatcher struct {
	filter userFilter
	folded string
}

func newMatcher(f userFilter) userMatcher { return userMatcher{filter: f, folded: fold(f.Q)} }

func (m userMatcher) search(row userRow) bool {
	for _, text := range []string{row.Login, row.Email, row.UserID} {
		if strings.Contains(fold(text), m.folded) {
			return true
		}
	}
	return false
}

func (m userMatcher) matches(row userRow, member *identity.MemberInfo, now time.Time) bool {
	f := m.filter
	switch {
	case f.Q != "" && !m.search(row),
		f.Status != "" && row.State != f.Status,
		f.Membership != "" && membershipValue(member, now) != f.Membership,
		f.Role != "" && (member == nil || string(member.Role) != f.Role),
		f.Period != "" && (row.Membership == nil || row.Membership.RolePeriod != f.Period),
		f.Signin != "" && !slices.Contains(signinValue(row), f.Signin),
		f.Invitation != "" && row.InvitationExpires == "" && row.LinkPurpose != passwords.PurposeSet,
		f.Holder != "" && !row.Holder:
		return false
	}
	return true
}
