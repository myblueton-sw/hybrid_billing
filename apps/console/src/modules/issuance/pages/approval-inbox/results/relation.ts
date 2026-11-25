import { digest, exact, fail, freeze, id, json } from "./contract";
import type { ApplicationState } from "./history";
export type ApplicationContext = Readonly<{ kind: "initial" | "replacement"; position: "not_started" | "current" | "historical"; predecessor: string; predecessor_digest: string; predecessor_revision: number; predecessor_receipt: string; current_result: string; current_state: "none" | "unknown" | "recorded" }>;
export function applicationContext(raw: string, result: string, state: ApplicationState): ApplicationContext {
 const v = json(raw);
 exact(v, ["kind", "position", "predecessor", "predecessor_digest", "predecessor_revision", "predecessor_receipt", "current_result", "current_state"]);
 if (typeof v.kind !== "string" || typeof v.position !== "string" || typeof v.current_state !== "string" || !["initial", "replacement"].includes(v.kind) || !["not_started", "current", "historical"].includes(v.position) || !["none", "unknown", "recorded"].includes(v.current_state)) return fail();
 if ((v.current_state === "none") !== (v.current_result === "") || v.current_result !== "" && !id(v.current_result)) return fail();
 if (v.kind === "initial") {
  if (v.predecessor !== "" || v.predecessor_digest !== "" || v.predecessor_revision !== 0 || v.predecessor_receipt !== "") return fail();
 } else if (!id(v.predecessor) || v.predecessor === result || !digest(v.predecessor_digest) || !Number.isInteger(v.predecessor_revision) || Number(v.predecessor_revision) < 1 || Number(v.predecessor_revision) > 2147483647 || v.predecessor_receipt !== "" && !digest(v.predecessor_receipt)) return fail();
 if (v.position === "not_started") {
  if (state !== "not_started" || v.current_result === result || (v.kind === "initial" ? v.current_state !== "none" : v.predecessor !== v.current_result || v.current_state === "none" || (v.predecessor_receipt === "") !== (v.current_state === "unknown"))) return fail();
 } else {
  if (state === "not_started" || v.current_state === "none" || v.kind === "replacement" && !digest(v.predecessor_receipt)) return fail();
  if (v.position === "current" ? v.current_result !== result || v.current_state !== state : v.current_result === result || v.predecessor === v.current_result || state !== "recorded") return fail();
 }
 return freeze({ kind: v.kind, position: v.position, predecessor: v.predecessor, predecessor_digest: v.predecessor_digest, predecessor_revision: v.predecessor_revision, predecessor_receipt: v.predecessor_receipt, current_result: v.current_result, current_state: v.current_state }) as ApplicationContext;
}
export function sameApplicationContext(a: ApplicationContext, b: ApplicationContext): boolean {
 return JSON.stringify(a) === JSON.stringify(b);
}
