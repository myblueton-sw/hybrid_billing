---
type: VisualSpecification
title: Administration diagnostic feedback extension
description: Defines the scoped P01–P05 error-reference additions using the existing full-width feedback pattern.
status: stable
---

# Scoped items

| Item ID | Location and behavior | States |
| --- | --- | --- |
| P01:error:diagnostics | Installation row/form and read-error feedback | Refused, field error, unavailable, denied, session ended |
| P02:error:diagnostics | Integrations row/form and read-error feedback | Same plus unknown delivery, rate limit, cancelled confirmation and lost right |
| P03:error:diagnostics | Recovery row/form and read-error feedback | Refused, field error, unavailable, denied, session ended |
| P04:error:diagnostics | Retention normal/stale form, kept row and read-error feedback | Refused, field error, rate limit, unavailable, denied, session ended |
| P05:error:diagnostics | Evidence row/form and read-error feedback | Refused, field error, unavailable, denied, session ended |

# Presentation

Reuse the current neutral feedback region and definition list. Server code and server request ID always have an explicit known or unavailable value. Browser request and target references appear when actually known. Each retained failed action has a separately labelled Action ID so a row's precedence-selected explanation cannot misattribute another action's reference. These references do not prove completion. Existing field-location and action-specific next-step guidance remain.

Use the available page width, readable internal grouping, canonical local Plex fonts, square geometry and neutral separators. Tokens wrap within their cell and isolate LTR inside Arabic RTL. Do not add decorative coloured borders, copied assets or owner-acceptance marks. EN/KO/AR and1440/1280/1024/390 are the inspected variants.

# Evidence and review boundary

Microsoft's [notification-reference example](https://learn.microsoft.com/en-us/entra/identity/enterprise-apps/troubleshoot-password-based-sso#view-portal-notification-details), inspected2026-11-20, distinguishes browser request identity and notification identifiers. Adopt explicit attribution and readable grouping; reject decorative error-colour cues and unrelated identity-data collection. The existing product component supplies layout rather than importing a third-party screen. See the glossary for Korean evidence and the Arabic fallback limitation.

Independent read-only source and TW/render reviews passed for the actual final candidate. A Claude CLI ui-designer invocation was attempted with actual assigned renders and these references, but returned a weekly quota limit (resetNov24, Asia/Seoul); no Claude design or render approval is claimed. Native/practitioner and assistive-technology approval, actual Linux/Windows execution and owner FIX remain unverified. This scoped feedback extension is not a new screen composition or a release.
