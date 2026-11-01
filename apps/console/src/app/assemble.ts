import type { Module, Extension, Profile } from "../shared/types";
export function assemble(
  modules: Module[],
  extensions: Extension[],
  profile: Profile,
) {
  const unique = (ids: string[], kind: string) => {
    if (new Set(ids).size !== ids.length) throw Error(`duplicate ${kind}`);
  };
  unique(
    modules.map((m) => m.id),
    "module",
  );
  unique(
    extensions.map((e) => e.id),
    "extension",
  );
  unique(profile.modules, "selection");
  unique(profile.extensions, "selection");
  if (
    Object.keys(profile).some(
      (k) => !["id", "modules", "extensions"].includes(k),
    )
  )
    throw Error("unknown profile field");
  const byId = new Map(modules.map((m) => [m.id, m]));
  const active = profile.modules.map((id) => {
    const m = byId.get(id);
    if (!m || m.version !== 1) throw Error("unknown or incompatible module");
    return m;
  });
  const visiting = new Set<string>(),
    visited = new Set<string>();
  function visit(id: string) {
    if (visiting.has(id)) throw Error("dependency cycle");
    if (visited.has(id)) return;
    const m = active.find((m) => m.id === id);
    if (!m) throw Error("missing dependency");
    visiting.add(id);
    m.requires.forEach(visit);
    visiting.delete(id);
    visited.add(id);
  }
  active.forEach((m) => visit(m.id));
  const routes = active.flatMap((m) => m.routes);
  unique(
    routes.map((r) => r.id),
    "route",
  );
  const slots = active.flatMap((m) => m.slots);
  unique(slots, "slot");
  const added = profile.extensions.map((id) => {
    const e = extensions.find((e) => e.id === id);
    if (
      !e ||
      e.version !== 1 ||
      e.slotVersion !== 1 ||
      !active.some((m) => m.id === e.module && m.slots.includes(e.slot)) ||
      !slots.includes(e.slot)
    )
      throw Error("invalid extension capability");
    return e;
  });
  return {
    profile: profile.id,
    routes,
    extensions: added.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0)),
    modules: active.map((m) => m.id),
  };
}
