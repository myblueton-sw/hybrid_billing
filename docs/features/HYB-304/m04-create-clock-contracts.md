---
type: VerificationReport
title: HYB-304 create component authority at wall-clock boundaries
description: Defines quiesced native clock-boundary verification of service creation with genuine collected credential resources and exact proposed revisions.
status: draft
sources:
  - resource: ./m04-activation-rollback-contracts.md
---

# Reviewed scope and ownership

HYB-304 remains In Progress, accountable Seung Woo Park, machine mac_mini. Root owns new service_create_clock_fixture_test.go, service_create_clock_test.go and scoped evidence/documents. Separate read-only /root/hyb310_review approved the corrected bounded design. The initial design was rejected because direct Apply does not commit server revision and an expiring pending pickup may legitimately be pruned. The revised design uses actual collection before quiescence and distinguishes proposed from committed revisions.

Each fresh fixture uses an instance-owned mutex clock and independent monotonic elapsed time. Genuine HTTP password sign-in, service creation, credential issuance and original-session collection establish populated resources. Before full Close, snapshot canonical service/credentials/declaration, the genuine collected marker, zero pending pickups, delivery version, declaration count and server revision. Full Close joins owned workers/history; captured module verifier remains valid. Set future time to actual caller creation plus absolute lifetime plus one minute, within configured wall-lead tolerance. No global overrides, stack inspection or private registered-state edits are permitted.

Capture a genuine verifier before arming. Under the normal domain lock, set the dispatcher caller slot to that verifier, preserving exact previous caller/fault values for deferred restoration. Direct Apply receives complete valid create fields. Capture clock.finish immediately after Apply, before observers. A healthy cutoff-zero control derives the finite observation horizon; each cutoff from one through horizon plus one gets a fresh fixture and never resets an adopted jump. Clock counts select input boundaries, not per-query benchmarks.

Only this create path has a source-proven final configured-clock observation inside final identity CreateServiceUser admission, before untimed identity/declaration writes and Next. An observed expiry must return a mapped authentication/dependency refusal and empty outcome without new service/declaration/credential/pickup or committed revision changes. A no-jump success must return service_created, only declaration_version1 and the precomputed proposed Next revision; actual server revision remains unchanged because dispatcher commit is outside this component. Require canonical owner/creator/state, exact count increase and declaration version1; existing collected resources remain unchanged. A fresh supported Admin2 observer runs at adopted time after capture; authentication maintenance is outside unchanged business state.

Actual native profiles establish whether the remaining ServiceCount refusal arm was reached; no cutoff-to-function attribution is guessed. Relevant semantic mutations, restoration/private-cache cleanup, source/evidence review and regression/race/vet/portable checks remain planned. This is quiesced component execution, not live HTTP scheduling or command-ledger certification. Required100%, authorized merge/read-back and parent M03/M02 remain open.


# Initial execution and source review

The initial compile attempt failed because the test referenced a nonexistent serviceDeclaration.Service field. The fixture was corrected to use its actual keyed declaration lookup and canonical Organization/Purpose fields; no product code changed. The corrected scoped native sweep passed1.540s. Separate read-only /root/hyb310_review passed the actual source, including genuine collection, close/capture lifecycle, exact outcome and snapshot boundaries.

Both semantic mutants failed intended assertions: ignoring final identity insertion refusal allowed an outcome after observed expiry; committing the counter during direct Apply changed recorded server revision despite preserving its proposed outcome revision. Each restored its exact original SHA and removed the private cache. This proves the final refusal and component revision contracts, not independent necessity of every preceding authority check. Actual final verification follows.


# Final native measurement

Full server short passed199.241s at native99.8%,11,779 weighted statements/28 uncovered. Five named race groups passed17.250s; server vet and Linux/Windows amd64 test compilation passed. Foreign execution remains unverified. The separate focused native sweep passed1.813s and hits the single new ServiceCount refusal unit. Prior/full/focused profiles match production blocks and weights; no profile union or guessed cutoff-to-function attribution is used.

The [source/artifact manifest](create-clock-evidence.json) records source and actual artifacts. Direct Apply keeps committed revision unchanged and returns a proposed revision; it does not certify dispatcher commit, live HTTP scheduling or closed-history writes. Existing collected resources stay intact; authentication maintenance is excluded from unchanged business state. Prior75-package, frontend/browser and bounded aggregate evidence retain their source/run boundaries and were not rerun. Two documentation status checks and manual metadata/local-link checks are limited checks. The [remaining guard assessment](m04-coverage-residuals.md) records an unresolved owner quality criterion; no waiver is inferred. Required100%, authorized merge/main read-back and parent M03/M02 remain open.
