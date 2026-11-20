import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { guardUnload, leaveDeliberately, leaving, onDeliberateLeave, stayAgain, watchRestore } from "./unload-intent";

const fire = (handler: (e: BeforeUnloadEvent) => void) => {
  const e = new Event("beforeunload", { cancelable: true }) as BeforeUnloadEvent;
  handler(e);
  return e.defaultPrevented;
};

test("a deliberate leave silences the App and Reveal guards, and they guard again after it is cleared", () => {
  stayAgain();
  const app = guardUnload(() => true);
  const clean = guardUnload(() => false);
  assert.equal(fire(app), true);
  assert.equal(fire(clean), false);
  let released = 0;
  const stop = onDeliberateLeave(() => { released += 1; });
  leaveDeliberately();
  assert.equal(leaving(), true);
  assert.equal(released, 1);
  assert.equal(fire(app), false);
  stop();
  stayAgain();
  assert.equal(fire(app), true);
});

test("a back-forward cache restore clears the flag; a normal page show does not (L1)", () => {
  const listeners = new Map<string, (e: Event) => void>();
  const target = {
    addEventListener: (type: string, fn: (e: Event) => void) => listeners.set(type, fn),
    removeEventListener: (type: string) => listeners.delete(type),
  } as unknown as Pick<Window, "addEventListener" | "removeEventListener">;
  const stop = watchRestore(target);
  leaveDeliberately();
  listeners.get("pageshow")?.(Object.assign(new Event("pageshow"), { persisted: false }));
  assert.equal(leaving(), true);
  listeners.get("pageshow")?.(Object.assign(new Event("pageshow"), { persisted: true }));
  assert.equal(leaving(), false);
  stop();
  assert.equal(listeners.has("pageshow"), false);
});

const files = (dir: string): string[] =>
  readdirSync(dir).flatMap((name) => {
    const path = join(dir, name);
    return statSync(path).isDirectory() ? files(path) : [path];
  });

test("only explicit sign-out leaves deliberately, and a new session clears it (L1, L2)", () => {
  const root = join(import.meta.dirname, "..");
  const callers = files(root)
    .filter((p) => /\.(ts|tsx)$/.test(p) && !/\.test\.tsx?$/.test(p) && !p.endsWith("unload-intent.ts"))
    .flatMap((p) => (readFileSync(p, "utf8").match(/leaveDeliberately\(/g) ?? []).map(() => p.slice(root.length + 1)));
  assert.deepEqual(callers, ["service/Service.tsx"]);
  const source = readFileSync(join(root, "service", "Service.tsx"), "utf8");
  assert.match(source, /logout\(session\)[^\n]*\n\s*leaveDeliberately\(\);\n\s*location\.assign\(serviceUrl\("signin", l, "signed-out"\)\)/);
  assert.match(source, /if \(session\) stayAgain\(\);/);
});
