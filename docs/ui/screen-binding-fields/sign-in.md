---
type: Screen Binding Field Map
title: Field bindings of the sign-in screens L01 to L03
description: For each field, message and state of the service-mode sign-in screens L01, L02 and L03, the sign-in API read or call and JSON field it comes from, or the client-side derivation, with the console line that renders it.
status: stable
tags: [HYB-295, screen-binding, L01, L02, L03]
sources:
  - resource: ../screen-binding-register.md
  - resource: ../../features/HB-257/plan.md
  - resource: ../../features/HB-260/verification.md
---

# Sign-in field bindings

This is part of the [screen binding register](../screen-binding-register.md#sign-in-screens-l01l03), as `HB-257/plan.md:177-178` requires. It is based on main `82c9272`. Owner acceptance is pending for every screen here.

- **Short paths.** `Service.tsx`, `api.ts` and `copy.ts` are under `apps/console/src/service/`; `signin.go` is `apps/api/server/signin.go`.
- **Copy.** Every heading, label and message is fixed copy from `copy.ts` (`c.*`). It is not service data unless the Source column names a JSON field.
- **Typed values.** Values the person types are request fields, not read fields.

## Shared frame and session strip

| Area | Field or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Frame | Pre-launch notice | fixed copy `c.noticeLead`, `c.notice` | none | `Service.tsx:14-19`, used at `:43` and `:357` |
| Frame | Loading while the session is checked | derived: `session === undefined` before `GET /api/v1/session` answers | none | `Service.tsx:443` |
| Session strip | Workspace | `GET /api/v1/session` → `workspace` (`boundary.go:107`) | `api.ts:101` | `Service.tsx:359` |
| Session strip | User | `GET /api/v1/session` → `user` (`boundary.go:106`) | `api.ts:100` | `Service.tsx:360` |
| Session strip | Sign out | `POST /api/v1/auth/logout` | `api.ts:121` | `Service.tsx:361` |
| Session | `role`, `login`, `csrf_token` | `GET /api/v1/session` (`boundary.go:108-110`) | `api.ts:102-104` | not displayed; the CSRF token is sent on later requests |

## L01 Sign in

| Area | Field or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Status | "You have signed out" | URL parameter `status=signed-out`, set by the console after logout | none | `Service.tsx:186` |
| Status | Session ended | URL parameter `status=ended` | none | `Service.tsx:187` |
| Form | Workspace, email, password | typed; sent as `workspace`, `login`, `password` to `POST /api/v1/auth/login` (`signin.go:208`) | none | `Service.tsx:194-196` |
| Form | Missing-field errors | derived from empty fields | none | `Service.tsx:164-166`, summary `:191` |
| Result | Failed sign-in | any 401 from login; one message for every cause | `api.ts:31-37` (`failureOf`) | `Service.tsx:175`, `:191` |
| Result | Limited, with wait time | HTTP 429 and the `Retry-After` header | `api.ts:31`, `:60-63` | `Service.tsx:149` |
| Result | Unavailable or expired | 503, network failure, or `csrf_refused` | `api.ts:31-37` | `Service.tsx:150` |
| Result | Signed in | login response → session (`signin.go:199-203`) | `api.ts:99-105`, `:117-119` | the shell renders; `Service.tsx:453` |
| Gate | Service unavailable, with Try again | `GET /api/v1/session` failed | `api.ts:107-116` | `Service.tsx:449-450` |
| Busy | "Signing in" | derived | none | `Service.tsx:198` |
| Link | Can't sign in → L03 | fixed link | none | `Service.tsx:201` |

## L02 Set password or activate from a one-time link

| Area | Field or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Token | One-time token | URL fragment only, removed before any request | `api.ts:148` (`fragmentToken`) | read at `Service.tsx:380`, cleared at `:398` |
| State | Checking | derived while `POST /api/v1/auth/link` is pending | none | `Service.tsx:255` |
| State | Check failed, with Try again | link call failed, but not with `link_invalid` | `api.ts:31-37` | `Service.tsx:257-266` |
| State | Link invalid or used | `link_invalid` code, or no token | `api.ts:31` | `Service.tsx:268-277` |
| Title | Set, reset or activate | `POST /api/v1/auth/link` → `purpose` (`signin.go:338`, `:347`) | `api.ts:127-133` | `Service.tsx:327` |
| Account | Email being set | `POST /api/v1/auth/link` → `login` (`signin.go:347`); not shown for activation | `api.ts:132` | `Service.tsx:329` |
| Form (activation) | Sign-in email, contact email | typed; sent as `login`, `email` to `POST /api/v1/auth/redeem` (`signin.go:352`) | none | `Service.tsx:338-339` |
| Form | New password, confirm | typed; sent as `password` | none | `Service.tsx:342-343` |
| Form | Rule and match errors | derived (`passwordProblem`, `loginValid`, `emailValid` in `helpers.ts`) | none | `Service.tsx:295-307`, summary `:331` |
| Result | Server password-rule refusal | `password_form` code | `api.ts:33` | `Service.tsx:318`, summary `:331` |
| Result | Done | redeem succeeded | none | `Service.tsx:280-289` |

## L03 Recovery guidance

| Area | Field or state | Source | Parser | Rendered at |
| --- | --- | --- | --- | --- |
| Route | Help view | URL parameter `view=help` | none | `Service.tsx:442` |
| Body | Forgot password, attempts, restart sections | fixed copy (`forgotBody` at `copy.ts:195`) | none | `Service.tsx:217-219` |
| Aside | Self-service recovery not available | fixed copy | none | `Service.tsx:223` |

L03 has no API read or call.

## HYB-304 authenticated boundary follow-up

The tables above retain their original main82c9272 line references. This follow-up maps the current shared authenticated boundary by symbol rather than assigning those old line numbers to the changed source. L02/L03 bindings are unchanged.

| Area | Producer and interpretation | Consumer |
| --- | --- | --- |
| L01 ended status | Actual AdminClient HTTP401 before body parsing, or parsed semantic unauthenticated failure such as csrf_refused; ordinary permission_denied remains a permission failure | `Service.tsx`: root-owned `authenticatedClient`, layout subscription and credential-matched retirement clear sensitive app content in place and show the neutral sign-in-again status |
| Shared authenticated API | One client per root credential tuple; retirement is latched and late subscribers observe it | `context.ts`: provider `client` and `useAdminClient`; `admin.ts`: `subscribeAuthenticationLoss` and `send` guards |
| Password confirmation | Credential identity plus prompt generation and component request scope | `Service.tsx`: scoped gate/callbacks; `Reauthenticate.tsx`: rejects obsolete after-await/error/focus/close callbacks; a retired approval cannot dispatch a retry |
| Local refusal diagnostics | A retired client has no new server response/request/operation identifier | `admin.ts`: local unauthenticated error preserves empty diagnostics; actual remote errors retain their actual diagnostics |

M05 confirmation/authentication-loss scenarios and shared-root mounted scenarios are bound in [advanced evidence](../../features/HYB-304/advanced-evidence.json). Root retirement does not certify every authenticated route's layout, native-language review or owner acceptance.
