package server

import (
	"errors"
	"reflect"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

var errConfigurationUncertain = errors.New("configuration operation may have changed retained or module state")

type financialConfigurationEffect struct{ changed bool }

func (e *financialConfigurationEffect) mark() {
	if e != nil {
		e.changed = true
	}
}

func validConfigurationSetup(setup *financialConfigurationSetupContext) bool {
	if setup == nil {
		return true
	}
	_, valid := parseContractDay(setup.On)
	return valid && validPlanHash(setup.ScopeDigest)
}

func (a *financialIssuance) storeConfiguration(p financialIssuanceConfiguration, e *financialConfigurationEffect) {
	old, ok := a.configurations[p.Spec.ID]
	if ok && reflect.DeepEqual(old, p) {
		return
	}
	a.configurations[p.Spec.ID] = copyConfiguration(p)
	e.mark()
}

func configurationPublicError(err error, e *financialConfigurationEffect) error {
	if e != nil && e.changed {
		return errConfigurationUncertain
	}
	if errors.Is(err, inputauthority.ErrDenied) {
		return commands.Fail(commands.PermissionDenied)
	}
	return draftPublicError(err, nil)
}

func configurationReceipt(p financialIssuanceConfiguration) map[string]string {
	observation := "proposed"
	if p.State == "confirmed" {
		observation = "active"
	}
	row := configurationHistoryRow(p, observation)
	row["receipt"], row["contract"], row["current_eligibility"] = "historical_configuration", p.Spec.Contract, "unchecked"
	return row
}

func (a *financialIssuance) admitConfigurationReceipt(p financialIssuanceConfiguration) error {
	fields := configurationReceipt(p)
	fields["confirmer"] = strings.Repeat("x", 128)
	for _, key := range []string{"proposed_at", "approval_observed_at", "activated_at"} {
		fields[key] = strings.Repeat("9", 35)
	}
	fields["state"] = "confirmation_unknown"
	fields["module_observation"] = "proposed"
	if err := a.boundedDraftFields(fields); err != nil {
		return err
	}
	row := configurationHistoryRow(p, "proposed")
	row["confirmer"] = fields["confirmer"]
	row["state"] = "confirmation_unknown"
	for _, key := range []string{"proposed_at", "approval_observed_at", "activated_at"} {
		row[key] = fields[key]
	}
	raw, err := draftJSON(row)
	if err != nil {
		return err
	}
	if len(raw) > 4096 {
		return commands.Fail(commands.PrerequisiteFailed)
	}
	return nil
}
