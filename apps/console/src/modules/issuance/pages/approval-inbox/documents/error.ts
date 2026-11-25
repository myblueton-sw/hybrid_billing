export class DocumentError extends Error {
  constructor(readonly code: string, readonly effect: "none" | "unknown" = "none", readonly operation = "") { super(code); }
}
