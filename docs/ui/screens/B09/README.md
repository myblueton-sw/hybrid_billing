# B09 approval inbox and result evidence

The canonical screen remains B09 `/approvals` with pending, decisions, delegation, history and recognition keys. [HB-110 candidate design](../../../features/HB-110/approvals.md) maps stable implementation IDs to actual inbox/card/filter/error behavior; [verification](../../../features/HB-110/verification.md) distinguishes actual unconnected reads from isolated labelled synthetic QA. The original spec/design evidence is retained and is not certified fully implemented by this frontend increment.

The normal screen has no bound financial approval producer. Counts remain unknown; no request, actor, amount or successful decision is invented. Live responses fail closed pending separate reviewed producer integration. No approve/reject/delegation/posting command exists. Financial workflow, human/native review and owner FIX remain deferred. Every view retains the common revision-bound review control; automated QA never confirms owner acceptance.
