---
type: DesignPlan
title: A scope selector for the identifier "all" must not read as covering every one
description: The permissions read gives scope selectors in a structured form so that an identifier spelled "all" is not confused with a selector that covers every customer, legal entity or contract.
status: draft
---

# Scope and ownership

HYB-369, machine:mac_mini. Found in the review of the permissions read (HYB-304, PR 457).

# Facts

- The whole access read shows a selector as text: the identifier, or the word "all" when it covers every one (`apps/api/server/access.go` `selectorText`). The permissions read reused that text and its filters treated the stored text "all" as covering every one.
- The auth module does not reserve the identifier "all": `One("all")` is a valid selector for one customer. The service catalog relies on this and tests it (`modules/servicecatalog/catalog_test.go`, "explicit restrictions are not effective grants": a scope whose customer is the identifier "all" is not a scope for every customer).
- The request commands of M03 map the text "all" to the selector for every one, so no M03 command stores `One("all")`; a grant made from a catalog scope could carry it.

# Decision

The first plan, refusing the identifier in the auth module, was tried and dropped: it made the service catalog refuse a configuration it accepts on purpose, and two of its tests failed. The identifier stays valid.

The permissions read, which no screen uses yet, now gives each selector as `{"any": true}` or `{"any": false, "id": …}`, as the M04 reads do, and its `customer` and `legal_entity` filters match on that: a permission applies to the named one when its selector covers every one or names exactly it. The special filter value that listed only the permissions covering every one is removed; it was not asked for and was the ambiguous part.

# A decision that read the text back

The review of this change found one place where the stored selector's text was compared for a decision, not for display: `accessFacts.equivalent` (`apps/api/server/accessrequests.go`), which refuses a request to add a grant when an equivalent one exists. It compared `selectorText(stored)` with the request's customer text, where "all" means every customer. A stored grant for the customer named "all" was therefore taken as equivalent to a request for every customer, and that request was refused. `sameCustomer` now compares the selector itself: a request for every customer equals only a grant for every customer, and a request for one customer only a grant naming exactly it. The other readers of the request text (`pairRequest`, `applyAdd`) take it from the person's input and were already right.

# Not changed

The whole access read keeps its text form, where a customer named "all" and every customer look alike. That read's population fields are to be removed in increment C5 of the M03 design; until then the ambiguity there is as it was.

# Verification

The permissions tests include a grant scoped to a customer whose identifier is "all": it is listed under `customer=all` with the permissions that cover every customer, and not under `customer=c1`. A unit test of the equivalence check covers the four combinations of a stored selector and a requested customer. The source rule test and the API description test pass.
