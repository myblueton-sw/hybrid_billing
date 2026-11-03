---
type: Implementation
title: Planning registry and routing foundation candidate
description: Records the 49-screen registry, legacy routing and fail-closed audience contracts supporting the complete planning preview.
---

# Scope and ownership

HB74 owns the integrated planning-preview candidate. HB75 remains the tracking boundary for the registry/routing foundation; its full production action catalog is not declared complete by this preview. Root is sole implementation owner; accountable owner is Seung Woo Park. The single integration ticket branch starts at main 36394ed; subsequent changes remain under HB74, with child-ticket requirements linked rather than represented as separately completed work. The 49 records and 112 explicit redirect targets come from the reviewed HB72 registry. Module folders follow that registry's owning-module field. Stable English tab keys are assigned explicitly; legacy review identity is preserved. No owner FIX is copied.

This foundation exposes a pure resolver. HB74 integrates its preview shell and bodies; HB76–79 remain linked requirement scopes whose full production completion is not claimed. Scope keys define recognized query fields; screen-specific controls and invalidation remain the frame contract. Body specs remain in the authorized local workspace.

# Boundary design review

Independent read-only architect reviewed HB72 and local U03/M05/U02/B09 specifications: PASS for routing direction, permission completeness UNVERIFIED. Audience and permission context must precede body selection. Shared U03/M05 choose separate customer projections; M05 object segments cannot select another user. Fallback homes are checked for visibility, otherwise the resolver returns neutral denied. Assistant requires an additional permission flag. Internal evidence remains inaccessible to customer paths.

The review found that `specCommands` includes references and exclusions, not only executable actions, while some real actions appear only in prose. Therefore the registry preserves those references as unbound evidence, never executable grants. The view catalog is a preview contract only; delegation defaults closed. A complete source-backed action catalog, sensitivity rules and separation-of-duties policy remain pending independent review and cannot be inferred from role personas. No server authorization is implemented by this frontend foundation.

# Routing contract

The resolver accepts context separately from the hash. Query parameters do not select audience or confer permission. Valid legacy redirects keep query context. Canonical paths support object/tab segments and wizard steps. Malformed encoding, encoded separators, traversal, unknown tab names and excess segments fail to the audience's permitted home or denied state. Design tools are explicit admin-preview capabilities. Selection and cache invalidation are shell/frame responsibilities, not authorization by URL.

# Verification

Initial unit run: 52 tests passed, including 49 unique screens, all 112 redirects with language/mode/workspace retention, cross-audience rejection, safe home denial and object/step resolution. TypeScript passed. Actual change review and final checks are recorded below. No new visual composition is introduced in this foundation library; Claude CLI composition/render review belongs to the material shell/screen changes that consume it.

Final integration validation and independent review are recorded in [planning preview](planning-preview.md). The routing prototype-property regression is fixed with own-property lookup and the integrated unit suite passes 56 cases. Action-policy completeness remains unverified.
