package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	holdAllow          = "allow"
	holdRefuse         = "refuse"
	holdRoleDecrease   = "role_decrease"
	holdRevokeApproval = "revoke_approval"
	holdPerCommand     = "per_command"
	holdNoInvitations  = "allow_without_invitations"
	holdResetLinkOnly  = "reset_link_only"
	holdPeriodDecrease = "period_decrease"
)

type holdRule struct {
	rule   string
	reason string
}

const (
	reasonReduce   = "reduces access"
	reasonPending  = "withdraws or ends a pending proposal"
	reasonStop     = "stops a harmful or outbound effect"
	reasonPlace    = "places a hold"
	reasonIncrease = "adds or restores access"
	reasonChange   = "changes the installation, delivery, organization or business state"
	reasonTrust    = "the trust workflow is frozen until restart"
)

var holdTable = map[string]holdRule{
	"financial_charge.request":                     {holdRefuse, reasonIncrease},
	"financial_charge.approve":                     {holdRefuse, reasonIncrease},
	"financial_charge.reject":                      {holdAllow, reasonReduce},
	"document_authority.propose_console":           {holdRefuse, reasonChange},
	"document_authority.propose_human":             {holdRefuse, reasonChange},
	"document_authority.approve":                   {holdRefuse, reasonChange},
	"document_authority.reject":                    {holdAllow, reasonPending},
	"document_authority.withdraw":                  {holdAllow, reasonPending},
	"financial_request_permission.propose_console": {holdRefuse, reasonChange},
	"financial_request_permission.propose_human":   {holdRefuse, reasonChange},
	"financial_request_permission.approve":         {holdRefuse, reasonChange},
	"financial_request_permission.reject":          {holdAllow, reasonPending},
	"financial_request_permission.withdraw":        {holdAllow, reasonPending},
	"financial_permission.propose":                 {holdRefuse, reasonChange},
	"financial_permission.approve":                 {holdRefuse, reasonChange},
	"financial_permission.reject":                  {holdAllow, reasonPending},
	"financial_permission.withdraw":                {holdAllow, reasonPending},
	"financial_authority.propose":                  {holdRefuse, reasonChange},
	"financial_authority.confirm":                  {holdRefuse, reasonChange},
	"financial_authority.revoke":                   {holdAllow, reasonReduce},
	"financial_policy.propose":                     {holdRefuse, reasonChange},
	"financial_policy.confirm":                     {holdRefuse, reasonChange},
	"pricing_policy.capture":                       {holdRefuse, reasonChange},
	"pricing_policy.propose":                       {holdRefuse, reasonChange},
	"pricing_policy.approve":                       {holdRefuse, reasonChange},
	"pricing_policy.revoke":                        {holdAllow, reasonReduce},
	"pricing_authority.propose_console":            {holdRefuse, reasonIncrease},
	"pricing_authority.propose_human":              {holdRefuse, reasonIncrease},
	"pricing_authority.approve":                    {holdRefuse, reasonIncrease},
	"pricing_authority.reject":                     {holdAllow, reasonPending},
	"pricing_authority.withdraw":                   {holdAllow, reasonPending},
	"input.confirm_publish":                        {holdRefuse, reasonIncrease},
	"input.reject":                                 {holdAllow, reasonPending},
	"input_authority.propose_console":              {holdRefuse, reasonIncrease},
	"input_authority.propose_human":                {holdRefuse, reasonIncrease},
	"input_authority.approve":                      {holdRefuse, reasonIncrease},
	"input_authority.reject":                       {holdAllow, reasonPending},
	"input_authority.withdraw":                     {holdAllow, reasonPending},
	"identity.suspend_user":                        {holdAllow, reasonReduce},
	"identity.suspend_membership":                  {holdAllow, reasonReduce},
	"identity.depart_user":                         {holdAllow, reasonReduce},
	"identity.remove_membership":                   {holdAllow, reasonReduce},
	"identity.change_role":                         {holdRoleDecrease, "a change that neither promotes to administrator nor lengthens or removes the role end reduces access; anything else is an increase"},
	"identity.set_role_period":                     {holdPeriodDecrease, "a shorter role period reduces access; a longer, removed or restoring period is an increase"},
	"bulk_revocation.execute":                      {holdAllow, reasonReduce},
	"bulk_revocation.add_targets":                  {holdAllow, "stages a reduction and changes no access"},
	"bulk_revocation.add_matching":                 {holdAllow, "stages a reduction and changes no access"},
	"bulk_revocation.prepare":                      {holdAllow, "freezes a reduction and changes no access"},
	"bulk_revocation.discard":                      {holdAllow, reasonPending},
	"identity.revoke_invitation":                   {holdAllow, reasonReduce},
	"identity.create_user":                         {holdRefuse, reasonIncrease},
	"identity.invite":                              {holdRefuse, reasonIncrease},
	"identity.reinstate_user":                      {holdRefuse, reasonIncrease},
	"identity.reinstate_membership":                {holdRefuse, reasonIncrease},
	"identity.issue_set_link":                      {holdRefuse, "issues a credential path"},
	"identity.issue_reset_link":                    {holdRefuse, "issues a credential path"},
	"access.request_revoke":                        {holdAllow, reasonReduce},
	"access.revoke_delegation":                     {holdAllow, reasonReduce},
	"access.approve_request":                       {holdRevokeApproval, "approving a revoke reduces access; approving an addition is an increase"},
	"access.reject_request":                        {holdAllow, reasonPending},
	"access.withdraw_request":                      {holdAllow, reasonPending},
	"access.request_grant":                         {holdRefuse, reasonIncrease},
	"service.suspend":                              {holdAllow, reasonReduce},
	"service.retire":                               {holdAllow, reasonReduce},
	"service.revoke_credential":                    {holdAllow, reasonReduce},
	"service.create":                               {holdRefuse, reasonIncrease},
	"service.reinstate":                            {holdRefuse, reasonIncrease},
	"service.issue_credential":                     {holdRefuse, reasonIncrease},
	"recovery.lock":                                {holdAllow, reasonStop},
	"recovery.withdraw":                            {holdAllow, reasonPending},
	"recovery.approve":                             {holdRefuse, "could lift containment"},
	"recovery.propose_unlock":                      {holdRefuse, "could lift containment"},
	"retention.place_hold":                         {holdAllow, reasonPlace},
	"retention.stop_run":                           {holdAllow, reasonStop},
	"retention.withdraw":                           {holdAllow, reasonPending},
	"retention.approve":                            {holdRefuse, "could release a hold or start deletion"},
	"retention.propose_policy":                     {holdRefuse, "could release a hold or start deletion"},
	"retention.propose_release":                    {holdRefuse, "could release a hold or start deletion"},
	"integrations.plugin_stop":                     {holdAllow, reasonStop},
	"integrations.plugin_disable":                  {holdAllow, reasonStop},
	"integrations.subscription_pause":              {holdAllow, reasonStop},
	"integrations.subscription_revoke":             {holdAllow, reasonStop},
	"integrations.plugin_withdraw":                 {holdAllow, reasonPending},
	"integrations.subscription_withdraw":           {holdAllow, reasonPending},
	"integrations.subscription_withdraw_resume":    {holdAllow, reasonPending},
	"integrations.letter_withdraw_redeliver":       {holdAllow, reasonPending},
	"integrations.letter_withdraw_dismiss":         {holdAllow, reasonPending},
	"integrations.redelivery_reapprove":            {holdRefuse, reasonChange},
	"integrations.plugin_propose":                  {holdRefuse, reasonChange},
	"integrations.plugin_approve":                  {holdRefuse, reasonChange},
	"integrations.subscription_propose":            {holdRefuse, reasonChange},
	"integrations.subscription_approve":            {holdRefuse, reasonChange},
	"integrations.subscription_request_resume":     {holdRefuse, reasonChange},
	"integrations.subscription_approve_resume":     {holdRefuse, reasonChange},
	"integrations.letter_request_redeliver":        {holdRefuse, reasonChange},
	"integrations.letter_approve_redeliver":        {holdRefuse, reasonChange},
	"integrations.letter_request_dismiss":          {holdRefuse, reasonChange},
	"integrations.letter_approve_dismiss":          {holdRefuse, reasonChange},
	"installation.withdraw":                        {holdAllow, reasonPending},
	"installation.approve":                         {holdRefuse, reasonChange},
	"installation.propose_plan":                    {holdRefuse, reasonChange},
	"relationship.discard":                         {holdAllow, reasonPending},
	"relationship.propose":                         {holdRefuse, reasonChange},
	"relationship.activate":                        {holdRefuse, reasonChange},
	"view_group.discard_draft":                     {holdAllow, reasonPending},
	"view_group.draft":                             {holdRefuse, reasonChange},
	"view_group.edit":                              {holdRefuse, reasonChange},
	"view_group.propose":                           {holdRefuse, reasonChange},
	"contract.suspend":                             {holdRefuse, reasonChange},
	"contract.resume":                              {holdRefuse, reasonChange},
	"contract.renew":                               {holdRefuse, reasonChange},
	"contract.end":                                 {holdRefuse, reasonChange},
	"contract.withdraw":                            {holdRefuse, reasonChange},
	"contract.evidence":                            {holdRefuse, reasonChange},
	"contract.settle":                              {holdRefuse, reasonChange},
	"contract.reopen":                              {holdRefuse, reasonChange},
	"release_evidence.withdraw":                    {holdAllow, reasonPending},
	"release_evidence.approve":                     {holdRefuse, reasonChange},
	"release_evidence.retire_requirement":          {holdRefuse, reasonChange},
	"release_evidence.retire_profile":              {holdRefuse, reasonChange},
	"trust.propose":                                {holdRefuse, reasonTrust},
	"trust.validate":                               {holdRefuse, reasonTrust},
	"trust.approve":                                {holdRefuse, reasonTrust},
	"trust.activate":                               {holdRefuse, reasonTrust},
	"trust.reject":                                 {holdRefuse, reasonTrust},
	"trust.cancel":                                 {holdRefuse, reasonTrust},
}

var holdRoutes = map[string]holdRule{
	"/api/v1/commands/{command}": {holdPerCommand, "each command follows the command table"},
	"/api/v1/me/sessions/end":    {holdAllow, "ends sessions"},
	"/api/v1/me/password":        {holdAllow, "changes the caller's own password"},
	"/api/v1/me/reauthenticate":  {holdAllow, "password confirmation for the allowed step-up commands"},
	"/api/v1/auth/login":         {holdNoInvitations, "signs in to existing memberships only, so administrators can read and reduce"},
	"/api/v1/auth/logout":        {holdAllow, "ends a session"},
	"/api/v1/auth/link":          {holdAllow, "reads a link without using it"},
	"/api/v1/auth/redeem":        {holdResetLinkOnly, "a reset link only rotates an existing credential; a set link or activation adds an account"},
}

func (s *Server) held() bool { return s.hold != nil }

func (s *Server) enterHold(reason string) {
	if s.hold == nil {
		s.hold = &trustHold{reason: reason, since: s.clock()}
	}
	s.log.Error("trust_hold_entered", "reason", reason)
}

func (s *Server) holdAllows(command string, call commands.Call, caller identity.Verifier) bool {
	if command == "financial_charge.review" || command == "financial_charge.prepare" || command == "financial_request_permission.inspect" || command == "document_authority.inspect" {
		return true
	}
	if command == "financial_permission.inspect" || command == "financial_authority.inspect" || command == "financial_policy.current" || command == "financial_policy.inspect" || command == "pricing_policy.prepare" || command == "pricing_policy.inspect" || command == "pricing_authority.inspect" || command == "input_authority.inspect" || command == "input.inspect" {
		return true
	}
	rule, known := holdTable[command]
	switch {
	case !known:
		return false
	case rule.rule == holdAllow:
		return true
	case rule.rule == holdPeriodDecrease, rule.rule == holdRoleDecrease:
		return s.holdPeriodDecrease(caller, call, command)
	case rule.rule == holdRevokeApproval:
		n, ok := accesschange.Number(call.Target, accesschange.RequestPrefix)
		v, found := s.changes.Request(n, call.Answer.Workspace())
		return ok && found && v.Change.Kind == accesschange.ChangeRevoke
	}
	return false
}
