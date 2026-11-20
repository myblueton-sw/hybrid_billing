import type { CredentialRow, ServiceRow, ServicesPage, SelfPage } from "./admin";
import type { SelfEnd, SelfSession } from "./self-end";
import { subjectsOf } from "./access-read";
import { array, boolean, choice, integer, optional, positive, ReadParseError, shape, text } from "./strict-read";

const credential = shape<CredentialRow>({ id: text, not_before: text, not_after: text, state: choice("active", "expired", "revoked"), usable: boolean,
  secret_pickup: optional(choice("open", "collected", "never_collected")), secret_pickup_until: optional(text) });
export function credentialOf(value: unknown): CredentialRow {
  const parsed = credential(value);
  const { secret_pickup, secret_pickup_until, ...base } = parsed;
  if (secret_pickup_until !== undefined && (!secret_pickup_until || Number.isNaN(Date.parse(secret_pickup_until)))) throw new ReadParseError();
  if (secret_pickup === "open" && secret_pickup_until === undefined) throw new ReadParseError();
  if (secret_pickup === undefined && secret_pickup_until !== undefined) throw new ReadParseError();
  if (secret_pickup !== undefined && base.state !== "active") throw new ReadParseError();
  return { ...base, ...(secret_pickup === undefined ? {} : { secret_pickup }), ...(secret_pickup_until === undefined ? {} : { secret_pickup_until }) };
}
export const servicesPageOf = shape<ServicesPage>({ revision: text, now: text, owners: subjectsOf,
  services: array(shape<ServiceRow>({ id: text, state: choice("active", "suspended", "retired"), version: integer, creator: text, creator_login: text,
    owner: text, owner_login: text, credentials: array(credentialOf) })),
  limits: shape({ credential_days: integer, credentials_per_service: positive, services: positive, replay_seconds: positive }),
});
const session = shape<SelfSession>({ handle: text, current: boolean, signed_in: text, ends: text });
const end = shape<SelfEnd>({ operation: text, kind: choice("others", "one"), count: integer,
  state: choice("cleanup_running", "complete", "cleanup_incomplete"), blocked_at: text, list_generation: text, current_generation: text,
  generation_changed: boolean, digest: text, cleanup: shape({ done: integer, already_ended: integer, failed: integer, pending: integer }),
  sessions: array(shape({ handle: text, signed_in: text, result: choice("ended", "already_gone"), items: array(shape({ item: text,
    state: choice("done", "pending", "failed", "already_ended", "outside_scope", "not_applicable"), reason: (value) => value === undefined ? "" : text(value) })) })),
  not_included: array(shape({ handle: text, signed_in: text })), replay: boolean,
});
export const selfPageOf = shape<SelfPage>({ user_id: text, login: text, email: text, workspace: text, role: text,
  identities: array(shape({ method: text, login: text })), password_set: boolean, installation_administrator: boolean,
  sessions: array(session), sessions_generation: text, session_ends: array(end), idle_minutes: integer, max_age_seconds: integer,
  signed_in_at: text, authenticated_at: text, now: text,
});
