---
type: ImplementationPlan
title: Local preview sign-in autofill
description: Adds a development-only button for explicitly configured synthetic credentials.
status: stable
---

# Scope

[HYB-335](https://linear.app/myblueton/issue/HYB-335) follows the owner's request to click the local screen to fill email and password. Accountable owner: Seung Woo Park. Machine: mac_mini. Root owns implementation, QA and integration; separate reviewer checks the credential boundary and wording. Base: d7b6a3d. Required companion rules and authorized workspace inputs were verified from the local root workspace for integration.

# Behavior

The Vite development server receives explicit synthetic workspace/email/password values through process environment variables. Production builds and preview commands define no credential data, regardless of those variables. A loopback-only sign-in button fills the three existing fields, clears prior errors and focuses the sign-in button. It does not submit, create accounts, bypass authentication or store values in browser storage. Missing configuration hides the control. No credential value is committed.

Reuse existing entry-screen typography/control classes; no new layout or styling. EN/KO/AR labels describe filling local sample details, not authentication success. No owner screen acceptance is synthesized.

# Verification

Unit-test command/preview and loopback gates. Build with sentinel environment values and confirm none are in production output. Check actual local fill, keyboard activation, focus, no automatic request, error reset and successful normal sign-in. Render EN/KO/AR desktop/mobile. Review independently, then create PR; the owner’s standing instruction now authorizes reviewed, verified Git merges.

Independent design review: PASS with resolved-host guard required before credential injection. Final separate source/TW review and six-variant rendered review: PASS. Root owns implementation and evidence; reviewer made no edits.

# Current integration

Published candidate3e08f7f is merged with main f3ddf81 without rewriting history. Root owns the package test-list resolution and final read-back; the original feature source is unchanged. Preserve all HYB-327/330 test entries and add the local-preview configuration tests. Current review and execution evidence is recorded in [integration verification](integration.md).
