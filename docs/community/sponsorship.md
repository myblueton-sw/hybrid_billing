---
type: DesignDecision
title: Contributions and monthly sponsorship
description: Defines owner-approved sponsorship benefits and consent-based recognition for individuals, companies and organizations.
status: draft
---

# Scope and status

[HB-63](https://linear.app/hybrid-billing/issue/HB-63/implement-multilingual-whole-product-scalable-screen-redesign) adds the Contribute utility entry and `#community` route in the multilingual console. The owner accepted the following monthly benefit criteria; publication of tiers in GitHub Sponsors is unverified. This document and prototype do not change the live Sponsors account or process payment.

| Monthly USD | Tier | Benefits |
| --- | --- | --- |
| 5 | Supporter | Optional public recognition and development updates. |
| 25 | Backer | Supporter benefits plus roadmap opinions and surveys. |
| 100 | Team | Backer benefits plus priority review of feature/improvement requests and review results. |
| 300 | Organization | Team benefits plus development priority for adopted features and compatibility improvements. |
| 1,000 | Partner | Organization benefits plus roadmap discussion and discussion of agreed feature scope and development/application schedules. |

Individuals, companies and organizations may choose any tier. Sponsor identity type and benefit tier are independent; Team and Organization are tier names, not eligibility restrictions. No early-access benefit applies: development results enter the public source-available branch. Adoption depends on product fit, security and effort. Priority does not guarantee implementation or displace critical fixes. Sponsorship grants no commercial license, SLA, exclusive access or approval authority; [licensing policy](../licensing/product-policy.md) remains applicable. Update delivery and roadmap surveys are benefit definitions, not configured messaging integrations.

# Recognition placement

The contribution page contains monthly sponsorship, sponsors, and work contributors. Sponsor recognition is divided into individuals and companies/organizations. With separate listing consent, show an individual's chosen display name and approved profile link, or an organization's approved name, logo and website. Do not infer the organization from a payer's personal account or expose its representative. Logo permission must come from an authorized representative. Tier badges are optional and require explicit consent because they can disclose a funding level. Do not show exact payments or lifetime totals.

The current preview intentionally has no roster dataset and says not connected. It does not claim there are zero sponsors. No automatic API connection, admin editor or private sponsor record is implemented. Work contributions are credited separately using PR, issue or other accepted work evidence; money is not a prerequisite. The repository contributors link covers Git contributors, not all design, translation or other work.

# Initial manual workflow

1. The authorized account owner opens GitHub **Your sponsors → Dashboard → Your sponsors** and verifies current sponsorship. The dashboard is the financial source of truth; do not copy exports into this repository or the preview. [GitHub dashboard documentation](https://docs.github.com/en/sponsors/receiving-sponsorships-through-github-sponsors/viewing-your-sponsors-and-sponsorships).
2. Confirm the requested public identity (individual/company/organization), chosen display name, profile or website, and separate project-listing consent. For organizations verify authority to represent the company and use its logo. A public GitHub sponsorship alone is not assumed to authorize project logo reuse. Sponsors can change visibility in GitHub. [Visibility documentation](https://docs.github.com/en/sponsors/sponsoring-open-source-contributors/managing-your-sponsorship).
3. Only after verification, prepare a ticketed change containing approved public display fields. Keep consent evidence, contact details, private sponsor identities, amounts and credentials out of Git and browser payloads. No such records are added by this design change.
4. Review identity, link safety, logo permission, mobile/RTL layout and scope before publishing through the ordinary reviewed PR process. Never fetch authenticated sponsor records from browser code.
5. Handle correction/removal requests through the existing maintainer contact, and recheck current status before a roster update. Remove withdrawn listings promptly from current published artifacts; Git history may retain previously public values, so disclose this before collecting consent. Cancellation or loss of verified active status removes the current-sponsor listing; historical recognition needs separate consent. A future automated reconciliation service requires its own secret handling, visibility filtering, pagination and failure-state design.

The preview contact link opens a draft email only. No email, subscription, payment or external publication was performed. Native Arabic review and live benefit fulfillment remain pending.
