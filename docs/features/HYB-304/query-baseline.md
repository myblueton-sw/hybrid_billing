---
type: VerificationRecord
title: HYB-304 initial session-query regression baseline
description: Records actual missing-route failures before the approved M05 session-query implementation.
status: stable
sources:
  - resource: ./plan.md
---

# Executed baseline

Base main0f9d9a27d6202734e79d8f86ef6e65dfdfc5a752; root, machine mac_mini. The first increment's independent design review passed after two corrective rounds. This record does not claim source implementation or actual-change review completion.

`go test ./apps/api/server -run '^TestSessionQuery' -count=1` ran the three new semantic entry-point regressions against unchanged server source. All three failed as expected:

| Case | Actual baseline | Required behavior |
| --- | --- | --- |
| Bounded own other-session pages and no authentication-material disclosure | HTTP404 | HTTP200, one matching row per page and safe continuation |
| Missing current authentication | HTTP404 | HTTP401 before list disclosure |
| Foreign workspace query | HTTP404 | HTTP422 without rows/count |

The failing assertions are status checks on the real server handler; they are not absent-module import or compilation failures. Later page/content assertions have not yet executed, so no baseline or final PASS is claimed for them. The raw local log is `/private/tmp/hb304-query-before.log`; it contains no cookie, CSRF or credential value. Remaining cursor, authorization/time transitions, frozen selection, replay, resource accounting and UI cases must be added and verified with implementation.

No product source change, user runtime mutation or destructive test occurred in establishing this baseline. Existing server fixtures use isolated synthetic temporary data. This is fail-first evidence, not a release or feature-completion certificate.
