---
type: VerificationReport
title: HYB-304 service command dependency and staged conflict preservation
description: Defines component verification of execution-time dependency refusals and a supported public identity insertion before final create admission.
status: draft
sources:
  - resource: ./m04-reservation-input-contracts.md
---

# Scope and reviewed ownership

HYB-304 remains In Progress, accountable Seung Woo Park, machine mac_mini. Root owns service_command_dependency_test.go and scoped evidence/document updates only. Separate read-only /root/hyb310_review approved this design and the existing named service-declaration interleaving seam. No production edits or new hooks are planned.

Use genuine password sign-in and healthy HTTP creation/issuance of one populated service. Capture its public identity/credentials, declaration, retained pickup identity/deadline, directory version and server revision. Capture the legitimate verifier before acquiring the normal domain lock. During direct serviceCommand.Apply, temporarily set the dispatcher caller slot to that verifier, install the existing instance fault callback, then restore the exact previous callback/caller before unlocking through deferred cleanup. Post-step HTTP observations run after cleanup. Healthy resources remain unexpired; ordinary Apply pruning must not change the baseline.

Verify three contracts: declare execution with services unavailable refuses DependencyUnavailable without business effects; create allows its first named services check and refuses the second lookup with the same mapped code and exact two observations; create stages exactly one successful public Workspace.CreateServiceUser for its intended ID inside service-declaration before final insertion. That callback returns false. The outer create must refuse InvalidRequest specifically for name, return an empty outcome, preserve the exact competitor identity and increase count exactly once. It must publish no declaration, credential, pickup or server revision effect. Existing populated resources stay unchanged; the intentional identity insertion may change identity query generations.

This is direct command-component execution and staged public-module interleaving, not concurrent HTTP or full command-ledger certification. Callback insertion failure fails the fixture. Native scoped checks, relevant semantic mutation assertions with exact restoration/private-cache cleanup, actual-source review, full server regression, race/vet and portable compilation will be recorded honestly. Required100% and authorized merge/read-back remain pending; no coverage waiver or approval is inferred.


# Source review and semantic checks

Independent read-only /root/hyb310_review passed the actual three-contract implementation, including exact callback/caller restoration, public insertion timing and bounded state oracle. The initial scoped native run passed0.816s before explicit canonical Creator/Epoch premise tightening. The tightened focused native run passed0.823s; no initial-run claim is carried forward as final execution.

Three semantic mutations were assertion-killed: declaration lookup refusal removal changed the failure class; create lookup refusal removal allowed a success; conflict field attribution changed name to owner. Each matched the intended failure assertion, restored its exact product SHA and used a removed private cache. The conflict mutation proves field attribution, not universal adversarial interleaving safety. Five named race groups passed6.492s; server vet and Linux/Windows amd64 test-binary compilation passed. Foreign execution remains unverified. Full native server short passed195.198s at99.7%,11,779 weighted statements/30 uncovered. Prior/full/focused profiles have matching source blocks and weights; all 5 newly covered units are also hit by the separate focused contract profile. Profiles are not combined. Required100% remains unmet.


The [source/artifact manifest](command-dependency-evidence.json) binds this checkpoint. Prior75-package, Console/browser and bounded aggregate evidence remain separately attributed and were not rerun. Two documentation status checks plus manual metadata/local-link checks do not establish general semantic OKF certification. Authorized merge/main read-back and parent M03/M02 remain open.
