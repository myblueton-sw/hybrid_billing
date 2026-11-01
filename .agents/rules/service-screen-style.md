# Mandatory service-screen style constraints

The owner explicitly prohibited decorative colored borders in service screens. This is a MUST constraint, not an optional interpretation of brand tokens. Using canonical colors does not establish compliance.

- MUST NOT add brand/accent-colored edge strips to notices, colored selection outlines to cards, or decorative colored borders to tabs and panels.
- MUST use neutral separators, spacing, typography and explicit selection controls for hierarchy. A selected radio/checkbox, label, weight or underline must communicate state without a colored outline.
- MUST preserve visible keyboard focus. Focus indication is a transient accessibility state, not permission to add persistent decorative outlines.
- Before implementation, map these constraints to the affected components. Do not assume a copied baseline component is exempt from the owner's correction.
- During rendered verification, inspect normal, selected and feedback states at desktop/mobile sizes in KO/EN/AR. Verify actual computed borders as well as screenshots. Record the inspected scope; do not claim unchanged legacy screens were audited.
- A violation fails visual acceptance even if build, functional tests or token checks pass. Correct it before presenting the changed screen as compliant.

HB-67 entry regression checks cover notice/feedback edge removal, neutral selected-workspace borders and neutral active-tab borders. This coverage does not certify every legacy screen. Canonical logo geometry remains unchanged.

## Full viewport and external reference evidence — MUST

- MUST use the available viewport width as the service layout baseline. Do not silently add a centered, fixed maximum-width page container. Responsive columns and readable field/text grouping must work inside the full-width layout.
- MUST consult relevant external professional service screens or official service design guides before implementing or revising a screen layout. Record exact sources, the observed pattern, what is adopted, what is rejected and why. External patterns do not override owner instructions, existing IA or brand rules.
- MUST verify the rendered result against that evidence and the owner's constraints at desktop/mobile widths. A familiar convention, personal preference or token reuse is insufficient evidence.
- MUST retain neutral borders and explicit state indicators when adapting references. No new dependency or copied third-party asset is implied by research.
