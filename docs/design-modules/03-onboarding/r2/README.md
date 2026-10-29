---
type: Design
title: Minimal-action multi-provider FOCUS onboarding
description: Defines a common onboarding interface with provider-specific data routes, schema validation, coverage and separate evidence acceptance.
status: draft
---

# Scope and decision

[HB-49](https://linear.app/hybrid-billing/issue/HB-49/document-minimal-action-aws-focus-onboarding). Owner priority: FOCUS-first onboarding, starting with AWS, with the fewest user actions and a simple interface. Root owns this module revision and the baseline index link. Architect and ui-ux skills apply; a separate read-only reviewer checks security and financial-readiness boundaries. This is a design proposal, not a deployed connector or CloudFormation template. HB-51 adds the synthetic interactive review described below.

Baseline: frozen v10 / module 03-onboarding r1, pages 11–18. This r2 elaborates setup-connectors, setup-readtest, setup-schedule, setup-first, setup-readiness and setup-progress with AWS-specific child states. setup-profile and setup-attribution retain their existing obligations. Related modules: 02-access for authorization, 05-sources for source evidence, 06-contracts/07-reconcile for financial readiness. ON-01–05, BL-07 and EX-10 remain unchecked until runtime acceptance. [Research and limits](research.md); [review](review.md).

**Default experience: connect AWS → approve in AWS → return to a live collection status.** Three surfaces are not three clicks. Hide configuration mechanics, not account scope, resource creation or unresolved data quality.

# Multi-provider interface expansion (HB-51)

The owner clarified that this module must cover all providers, and that actual data integration and format validation are central. [HB-51](https://linear.app/hybrid-billing/issue/HB-51/document-multi-provider-focus-onboarding-interface-and-data-validation) updates this same draft rather than creating another competing revision. The AWS specification below remains the detailed AWS profile, not a universal provisioning recipe. The current proposal is not yet fixed as final and does not delete unreplaced v10 scope.

Common surfaces: choose source → review only missing connection setup → follow acquisition and validation. Supported source categories are AWS, Azure, Google Cloud, OCI, Alibaba Cloud, IBM Cloud, VMware/on-prem and existing file/storage. Native FOCUS availability and a verified Hybrid Billing connector are separate. Provider-specific setup cannot be presented as a universal OAuth approval. See [provider profiles and official evidence](provider-profiles.md).

The Korean interactive review [source](prototype.html) presents these surfaces using synthetic inputs only. It never reads files, queries cloud APIs, deploys resources or confirms billing evidence. Its preview-only state selector exposes first-arrival waiting, schema error, missing account and verification-summary states. It is outside the depicted product journey; do not count it as a production input. The preview applies the owner's Korean screen requirement while design/evidence documents stay English.

## Data and format validation contract

- Before access: bind the authenticated installation, source custodian, authorized accounts/resources, billing scope, operating identity and declared source profile. New accounts in a payer export require authorization, not automatic tenant membership.
- Source receipt: preserve raw files or bounded query result snapshots and query/response metadata, source location, complete manifest/page set, execution watermark, period and immutable revision. Object-store file, BigQuery result and paginated API have different receipt mechanisms. Scope and query budgets must be approved before data access.
- Source identity: pin provider dataset, declared FOCUS version, provider schema version and extensions, query/filter configuration, account scope and source schema digest. A matching header alone is not a trusted version/profile. Unknown profiles block accepted ingestion with a specific owner/action.
- Conversion: retain native versus converted provenance, target schema/version, exact adapter/mapping revision and unsupported functions. No LLM-generated mapping becomes active without controlled review. VMware requires metering history plus approved cost/allocation policies; it is not a CSP invoice.
- Field inspection: show source field, source type/unit, target field/version, transformation and pass/fail/unverified result. Preserve decimal precision, NULL versus zero, currency, quantities/units, timezone and interval boundaries. Keep BilledCost and EffectiveCost separate. Provider extensions remain attributable to originals.
- Full validation: examples help explain errors but acceptance covers every required file/page, row, authorized account and expected period, with independent count/identity and unit/currency/basis comparison. Show missing, partial, unsupported, duplicate, corrected and evidence-backed zero distinctly.
- Publication: received, schema-valid, coverage-complete, comparison-complete and operator-confirmed are separate states. Protected operator inspection does not grant customer visibility or financial acceptance. Current authority, revision and required comparisons gate input.confirm_publish; financial policy/amount/issuance approvals remain subsequent distinct controls.

The prototype field table illustrates common FOCUS concepts, not a certified provider column inventory. VMware uses a separate metering/policy illustration. Unknown file profiles, VMware and Alibaba analysis-only FOCUS cannot reach monetary acceptance through the generic successful example. Alibaba requires separately collected official bill overview/details for reconciliation and settlement. Contract and scope differences are visible before setup; deeper parameters remain collapsed. Evidence summaries link to existing setup-attribution, sources and reconciliation responsibilities rather than asking for the full financial configuration again.

## Provider-neutral acceptance additions (runtime NOT_RUN)

- M01: Each source shows its actual storage/API/query route, version/preview limits and the appropriate missing inputs; no generic cloud login silently substitutes for billing scope.
- M02: Unsupported Azure scope or unavailable provider preview entitlement blocks before creating resources; native-format availability is never inferred from a provider logo.
- M03: BigQuery/service-account location and query permissions, OCI report policy, OSS read scope and IBM full-page/month coverage are diagnosed separately. No key or token is entered into the mockup.
- M04: Unknown version/header/schema drift is held, decimal/NULL/currency units preserved, and conversion support is explicit.
- M05: Missing accounts, missing chunks/pages, duplicates and changed source snapshots never pass because the preview sample passed.
- M06: Human confirmation is unavailable while required scope, source, pricing basis or independent reconciliation evidence is missing. No inherited cross-customer authorization.
- M07: Switching providers or switching existing/new paths clears obsolete setup context and cannot reuse another provider's diagnostics. Keyboard focus moves to a visible control on explicit screen navigation.
- M08: Every simulated state remains labeled as example data; user interaction performs no external access or billing action.

# Deployment-realistic setup expansion (HB-52)

The owner requires actual operable configuration screens, not summary cards. [Conditional workflow](setup-workflow.md) and [provider guides](setup-workflow.md#provider-guides) now define inputs, owners, prerequisites, recovery and release gates. The earlier three-surface prototype is incomplete for this requirement. The one-account AWS path below applies only after every installation/default-profile prerequisite is verified; it is not the general setup experience. The [detailed configuration prototype](connection-setup.html) contains seven operable surfaces with provider-specific forms, conditional setup, administrator handoff and simulated diagnostics. It does not claim live provider calls. The earlier prototype.html remains the distinct compact navigation review, not the detailed configuration screen.

# Stack-first setup and console comparison (HB-53)

[Stack-first design and screenshot provenance](console-comparison.md) correct the default AWS experience: account/custody/installation prerequisite → generated Stack plan → customer AWS approval → automatic result inspection. Role/bucket/export inputs remain in existing/advanced paths. The detailed prototype now includes an optional official-console comparison panel. Historical examples and missing screenshots are explicit; no live template launch or complete current-console coverage is claimed.

**Owner review status:** keep the current functional draft; detailed visual refinement is deferred. Match the product-wide look and feel without the perceived Claude-style presentation, and later design guidance around the customer working in the actual provider console. Reference screenshots alone are insufficient. See [recorded feedback](console-comparison.md#owner-feedback-keep-this-draft-and-defer-visual-refinement).

# Preconditions that the product must not hide

The installation administrator first configures a customer-controlled workload identity able to request short-lived AWS role sessions. AWS-hosted workload roles are one candidate; an off-AWS installation needs a separately verified federation/credential-provider setup. There is no vendor-operated central AWS account, no collection of browser AWS credentials and no pasted access-key pair in this flow. An installation lacking this prerequisite shows a targeted administrator handoff before launch. The short path does not pretend this one-time installation work is free or already implemented.

The session must already have current collection authority for the selected installation/source custody and explicit permitted AWS account scope. An account ID entered in a form, successful AssumeRole, shared company name or newly observed linked account does not grant product authorization. A management-account export can contain many customers; its complete raw custody must be approved for the collecting operator before download. Customer attribution is separately approved, effective-dated and never inferred from labels. If custody or assignments are unknown, save a connection draft and route to setup-attribution; do not download into an arbitrary customer tenant.

This is an early **technical connection draft**, not a reorder or waiver of OP-05 financial onboarding. Legal entity, currency/time/period, attribution, issuance/ERP, pricing, FX/tax and approvals retain their prescribed financial sequence. Users can see connection progress while those tasks remain open; no rates, tax or issuer are guessed to shorten the wizard. No automatic customer access or financial eligibility results.

# Three primary surfaces

| Surface / stable child ID | What the user sees | User action | Automatic work |
| --- | --- | --- | --- |
| Connect / setup-connectors.aws-connect | AWS account ID, inherited authorized source owner/scope, short resource and charge summary; existing export as a secondary link | Enter or confirm one account ID; choose Connect AWS | Validate account format and authorized scope; reuse an existing same-scope connection; bind setup attempt, template version and workload principal |
| Approve / setup-connectors.aws-approval | AWS Quick create stack page in a new tab; original app remains at Waiting for AWS approval | Sign in or switch account if needed; review stack, acknowledge IAM creation, create stack; return to app | Prepopulate nonsecret parameters; resume bounded checks in the app without a manual Test connection button |
| Progress / setup-first.aws-progress | AWS setup checked / AWS preparing data / Checking received data / Review required, plus last check and next action | No action while waiting; leave and resume freely | Verify actual role/export/read evidence, detect source delivery, validate manifest and all chunks, preserve originals and independently compare normalized results |

Only a fully verified default-profile path can omit bucket, prefix, region, role and schedule entry from the first surface. Otherwise the actual provider configuration form must expose the missing fields before setup; HB-52 supersedes any universal single-field interpretation. Show one primary button and one secondary existing-export route. Advanced settings reveal the pinned profile and allow approved alternatives; changes invalidate dependent evidence. Account scope stays visible, even with one authorized account.

The common-path target, excluding entry navigation: one account-ID entry (zero when already known), one app launch, one AWS IAM acknowledgement, one AWS Create stack action and a return to the app. AWS login/MFA/account switching and any required installation setup are additional actions, measured separately. Exact clicks are NOT_RUN until tested against the supported AWS console. Later eligible publication requires a separate authorized confirmation; it is not hidden in the connection click.

# AWS preparation and return mechanism

Proposed standard profile: AWS commercial partition, supported Data Exports/CloudFormation region verified at release, native FOCUS 1.2 table FOCUS_1_2_AWS, all required/provider-extension columns, HOURLY detail, Parquet, a dedicated customer-owned bucket/prefix, CREATE_NEW_REPORT delivery revisions and recurring provider refresh. Current-month intake starts first; only evidenced available historical periods can be requested later. This is a proposed profile, not proof the current local consumed-field validator supports the complete AWS schema. China/GovCloud, alternate regions, centralized delivery, post-delivery KMS encryption and legacy exports require explicit supported profiles rather than silent fallback.

A versioned, reviewed template creates the scoped reader role, dedicated storage and its export-writer policy, then the export. Resource names/paths derive from the installation connection ID and account; deployment validates the expected account/partition. The installation supplies an exact customer-controlled trusted workload principal and unique connection ExternalId, not a broad vendor principal. ExternalId is a binding condition, not a secret or login credential. Reader authority is limited to intended export metadata and manifest/data objects under the approved prefix, plus required scoped decryption only when that encryption profile is supported. No collector CreateExport, IAM mutation, S3 write/delete or general administrative permissions.

AWS deployment uses the customer's authorized setup session. Setup permissions, export-service writing and collector reading remain separate. After stack completion, verify and record removal or expiry of temporary setup grants/session capabilities; if evidence is unavailable, show cleanup verification pending rather than claim completion. Pre-existing customer administrator privileges remain customer-managed, never silently revoked. Ending setup never removes the export writer. The review page discloses resources, account, scope and potential storage/request charges before creation. No automatic deletion of an existing stack, export or data. Disconnect stops collection and revokes reader access through an explicit customer action; retained evidence follows existing retention rules.

A Quick create link requires an AWS-accessible S3 template URL. The installation must provide an approved pinned artifact URL, or the administrator uses a downloaded template in CloudFormation. A GitHub HTML page or localhost is not a valid substitute. This artifact distribution is a release prerequisite, not a SaaS ingestion service. No secrets in URL parameters, stack outputs or logs; restrict configurable destinations and redirects under the existing endpoint policy.

No custom callback Lambda, automatic browser credential transfer or fictional OAuth return is required. Before launch the backend knows expected account, deterministic role name, attempt ID, template digest and ExternalId. On return/tab focus the authenticated app resumes polling using its workload identity. An ARN is only a lookup hint: role session identity, export configuration and actual scoped read must agree with the bound attempt. AWS console output alone cannot mark success. Wrong account, changed principal, expired attempt or changed template invalidates affected diagnostics. Without permission to inspect stack status, say Approval not yet verified rather than inventing a CloudFormation status; offer a link for the authorized AWS administrator.

If a deployment fails or its outcome is unknown, inspect the existing bound attempt/resources before retry. Do not create another export or bucket on every click. Retry concurrency is locked per installation/source/profile; only an explicit new attempt after resolution can provision again.

# Existing export without disruptive changes

The secondary Use existing FOCUS export path avoids duplicate exports and quota pressure. Reuse a previously verified reader connection when available. Otherwise establish a reader-only role first, with an administrator-approved export metadata scope. Discover only that approved scope; choose the sole compatible export automatically and visibly, or ask for one selection when multiple match. Where discovery permission is unavailable, accept one export ARN and request only the additional destination authorization actually needed. Reading export metadata does not authorize reading arbitrary bucket contents.

Keep existing bucket, encryption, query and refresh settings intact. Existing filtered or unsupported-version exports receive a precise limitation; never rewrite them or call them complete. A 1.0 source retains its declared version and requires an approved adapter or a separate user-approved 1.2 export. Quota exhaustion leads here; it never deletes another export. Advanced paths may require extra administrator work and do not inherit the default action-count promise.

# Progress, validation and completion

| State / setup-first.aws-progress | Meaning and primary action |
| --- | --- |
| Waiting for approval | No verified role/read yet. Resume AWS setup or hand off only this task. |
| AWS setup checked; preparing first delivery | Identity, export metadata and permitted path-list access verified; first object GetObject read remains pending until files arrive. Listing is not proof of an operational object read. Provider has not delivered files. No cost amount and no fake percentage. AWS may take up to 24 hours to begin delivery; show elapsed time and last check, not a completion guarantee. |
| Delivery delayed / action required | Past expected arrival or denied read. Distinguish generation, role, bucket/KMS, network and unsupported-period causes; one actionable retry after repair. |
| Validating delivery | Show observed files/rows and validation stages; percentage only with a reliable manifest denominator. Partial chunks never produce a completed bill. |
| Received; review required | Protected operator evidence is available, with account/period coverage, version, currencies and discrepancies. This is not an accepted customer cost view. |
| Eligible input published | Only after complete independent validation and explicit authorized input confirmation. All required source/billing-basis comparisons must already pass before eligibility. Later pricing-policy approval, customer amount approval and customer invoice issuance remain separate gates. |

Auto-checks pin the source table/version, manifest declaration, query/configuration, extensions, target conversion and capability profile. Detect a mismatch instead of guessing from a familiar column name. Preserve original files and source revisions. FOCUS conformance, the local consumed-field checks and billing eligibility are different claims; show known conformance gaps and affected functions. Missing data, missing permission and genuine evidence-backed zero are distinct.

Read one complete manifest execution and all its chunks. Verify object identity and stability around the read; changing manifests/objects restart the bounded read without mixing executions. Store immutable originals before transformation within authorized custody. Repeated monthly deliveries replace the same economic window as a new revision; they are not additive usage. CREATE_NEW_REPORT preserves source executions but does not make each a new charge. Existing overwrite sources need guarded reads and preserved internal revisions. Late corrections follow the existing reopen/confirmation contract.

Expected account/period coverage comes from independently approved scope, not merely accounts observed in the file. Newly appearing accounts are held for authorized attribution; management-account completeness is never assumed from a member export. Explain organization join/leave history gaps and missing periods. Cross-validation covers all declared rows/accounts/windows using independently derived counts, identities and unit/currency/basis totals. A checksum or summing the same transformed output twice is insufficient. Separate BilledCost, EffectiveCost and customer selling values; do not combine currencies or force amortized cost to equal a current invoice.

An operator-only receipt preview is visibly unaccepted, scope-filtered and separate from published customer queries. Eligible publication follows existing input.confirm_publish authority, full required reconciliation evidence and atomic revision recheck; missing comparison evidence remains blocked, even if the screen is simple. Changed source/profile/mapping/scope invalidates confirmation. Human confirmation cannot waive a blocker. Financial approvals are not replaced by this input confirmation.

Polling uses bounded backoff and persisted checkpoints; the current proposed policy is a short setup-check window followed by background delivery checks, with bounded concurrency and an explicit delayed state. Exact intervals and retention are release-profile settings requiring operational validation, not user wizard inputs. Closing the browser does not cancel server work. Permission revocation stops new reads/jobs and invalidates pending publication. Notifications stay in-app by default; external email requires separately configured consent and delivery support.

# Screen composition and copy intent

Use the v10 visual language and a compact main panel. Header: Connect AWS costs. One-line purpose: Connect FOCUS data to start checking your cloud costs. Keep source owner and authorized account scope above the main action. Show the account-ID field only when absent. Under Connect AWS: Creates a cost export and read-only collection access in your AWS account; storage and request charges may apply. Secondary: Use an existing export. Technical settings remain collapsed.

On progress, keep a persistent account label, text status with icon, last successful check and a safe next action. Waiting copy: AWS is preparing your first file. You can leave this page; we will continue checking. No mandatory success modal, copied role ARN, repeated test buttons or configuration checklist to click through. Review required links directly to the operator evidence summary, not a green Billing ready badge.

On receipt, show only useful summary fields: observed/expected accounts, available period, latest source refresh, FOCUS/profile version and blocking findings. Missing values display Unknown/Waiting, never zero. Do not show raw shared-payer procurement values to a customer viewer. Keyboard focus follows the returned setup status; announcements are concise and do not interrupt repeatedly. Errors are not color-only and retain previous valid inputs.

# Acceptance scenarios (all runtime NOT_RUN)

- A01: Known authorized account, identity and template ready: no technical text fields; approved action budget measured including AWS-side actions.
- A02: Missing workload identity or account custody: targeted admin/attribution handoff, no secrets collected and no raw download.
- A03: Wrong AWS account/partition, role or template changed: no green success; affected checks rerun after repair.
- A04: First delivery absent through 24 hours, browser closed/reopened: persisted progress, no zero amount, delay diagnosed without duplicate setup.
- A05: Existing export, two matches, quota exhausted, filtered export or FOCUS 1.0: minimal selection/repair, no mutation or false conformance.
- A06: AccessDenied, KMS failure, invalid endpoint and revoked membership during background work: scoped errors, no continued reads or secret disclosure.
- A07: Shared payer with unassigned/new member and cross-tenant viewer: no authorization from discovery; custody and row/field boundaries enforced.
- A08: Missing chunk, changing manifest, duplicate refresh and late correction: no partial completion, double counting or mixed revision.
- A09: Independent totals disagree, invoice basis missing, unsupported fields or unknown zero: preview held; confirmation cannot publish.
- A10: Complete evidence and authorized confirmation, then a racing new revision: stale decision rejected; retries do not duplicate publication.
- A11: Failed stack or unknown launch response, repeated Connect: resolve same attempt; no orphan proliferation or automatic data deletion.
- A12: Member-only export and organization movement: explicit limited account/history coverage, no organization-wide claim.
- A13: Keyboard/mobile/return focus and long wait: one clear next action, no polling focus theft; optional detail remains reachable.
- A14: Existing v10 financial setup incomplete: connection draft may progress only within custody; no silent financial defaults or issuance permission.

# Delivery boundary

Next: obtain owner feedback on the Korean interactive r2 review, complete the corresponding English visual review, then implement/test provider-specific adapters and setup templates under their own tickets. Live AWS deployment, account credentials, exact IAM policy execution and console action count have not been tested here. The frozen v10 book and registry are unchanged; r2 is not an assembled replacement edition.
