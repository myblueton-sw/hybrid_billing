import type { OrgHistory, OrgMember, OrgParty, OrgTarget } from "../../../../service/admin";

export function label(name: string, id: string): string {
  return name || id;
}

export type Progress = { complete: boolean; cursor: string };

export function loaded(id: string, p: Progress): boolean {
  return p.complete || id <= p.cursor;
}

export type NodeKind = "legal_entity" | "internal_unit" | "account" | "unknown";
export type ParentSource = "relationship" | "registry" | "none";
export type TreeNode = { id: string; name: string; kind: NodeKind; state: string; source: ParentSource; children: TreeNode[] };

const order: Record<NodeKind, number> = { legal_entity: 0, internal_unit: 1, account: 2, unknown: 3 };

function byKindThenName(a: TreeNode, b: TreeNode): number {
  return order[a.kind] - order[b.kind] || a.name.localeCompare(b.name) || a.id.localeCompare(b.id);
}

export type Conflict = { node: TreeNode; targets: OrgTarget[] };
export type Tree = { roots: TreeNode[]; unattributed: TreeNode[]; conflicts: Conflict[]; unloaded: TreeNode[]; cycles: TreeNode[] };

export function tree(parties: OrgParty[], hierarchy: OrgMember[], progress: Progress): Tree {
  const nodes = new Map<string, TreeNode>();
  for (const p of parties) nodes.set(p.id, { id: p.id, name: label(p.name, p.id), kind: p.kind, state: p.state, source: "none", children: [] });
  const parentOf = new Map<string, string>();
  const member = new Set(hierarchy.map((m) => m.member));
  const unattributedIds: string[] = [];
  const conflictList: { id: string; targets: OrgTarget[] }[] = [];
  const unloaded: TreeNode[] = [];
  const held = new Set<string>();
  for (const m of hierarchy) {
    if (!nodes.has(m.member)) nodes.set(m.member, { id: m.member, name: label(m.name, m.member), kind: "account", state: "", source: "none", children: [] });
    if (m.targets.length > 1) { conflictList.push({ id: m.member, targets: m.targets }); held.add(m.member); continue; }
    if (m.targets.length === 0) { unattributedIds.push(m.member); held.add(m.member); continue; }
    const target = m.targets[0];
    parentOf.set(m.member, target.id);
    (nodes.get(m.member) as TreeNode).source = "relationship";
    if (!nodes.has(target.id)) nodes.set(target.id, { id: target.id, name: label(target.name, target.id), kind: "unknown", state: "", source: "none", children: [] });
  }
  for (const p of parties) {
    if (member.has(p.id)) continue;
    if (!loaded(p.id, progress)) { unloaded.push(nodes.get(p.id) as TreeNode); held.add(p.id); continue; }
    if (!p.parent) continue;
    parentOf.set(p.id, p.parent);
    (nodes.get(p.id) as TreeNode).source = "registry";
    if (!nodes.has(p.parent)) nodes.set(p.parent, { id: p.parent, name: p.parent, kind: "unknown", state: "", source: "none", children: [] });
  }
  for (const [child, parent] of parentOf) nodes.get(parent)?.children.push(nodes.get(child) as TreeNode);
  for (const node of nodes.values()) node.children.sort(byKindThenName);
  const roots = [...nodes.values()].filter((n) => !parentOf.has(n.id) && !held.has(n.id)).sort(byKindThenName);
  const seen = new Set<string>();
  const walk = (n: TreeNode) => {
    if (seen.has(n.id)) return;
    seen.add(n.id);
    n.children.forEach(walk);
  };
  roots.forEach(walk);
  for (const id of held) {
    const n = nodes.get(id) as TreeNode;
    n.children.forEach(walk);
  }
  const cycles = [...nodes.values()].filter((n) => !seen.has(n.id) && !held.has(n.id)).sort(byKindThenName);
  for (const n of cycles) n.children = [];
  unloaded.sort(byKindThenName);
  const unattributed = unattributedIds.map((id) => nodes.get(id) as TreeNode).sort(byKindThenName);
  const conflicts = conflictList.map((c) => ({ node: nodes.get(c.id) as TreeNode, targets: c.targets })).sort((a, b) => byKindThenName(a.node, b.node));
  return { roots, unattributed, conflicts, unloaded, cycles };
}

export type Side =
  | { state: "target"; target: OrgTarget }
  | { state: "conflict"; targets: OrgTarget[] }
  | { state: "unattributed" }
  | { state: "none" }
  | { state: "unloaded" };
export type AccountRow = { id: string; name: string; unit: Side; payer: Side; entryFrom: string };

function side(m: OrgMember | undefined, id: string, progress: Progress): Side {
  if (!m) return loaded(id, progress) ? { state: "none" } : { state: "unloaded" };
  if (m.targets.length > 1) return { state: "conflict", targets: m.targets };
  return m.targets[0] ? { state: "target", target: m.targets[0] } : { state: "unattributed" };
}

export function applies(h: OrgHistory, on: string): boolean {
  return h.from <= on && (!h.until || on < h.until);
}

export function covering(history: OrgHistory[], on: string): OrgHistory | undefined {
  return history.find((h) => applies(h, on));
}

export function accounts(parties: OrgParty[], hierarchy: OrgMember[], payer: OrgMember[], on: string, hp: Progress, pp: Progress): AccountRow[] {
  const party = new Set(parties.map((p) => p.id));
  const units = new Map(hierarchy.map((m) => [m.member, m]));
  const payers = new Map(payer.map((m) => [m.member, m]));
  const ids = new Set([...units.keys(), ...payers.keys()].filter((id) => !party.has(id)));
  return [...ids].sort().map((id) => {
    const u = units.get(id);
    const p = payers.get(id);
    const unit = side(u, id, hp);
    return { id, name: label(u?.name || p?.name || "", id), unit, payer: side(p, id, pp), entryFrom: u && unit.state === "target" ? covering(u.history, on)?.from ?? "" : "" };
  });
}

export type Group = { id: string; name: string; members: { id: string; name: string }[] };

export function groups(members: OrgMember[]): { groups: Group[]; outside: OrgMember[] } {
  const byTarget = new Map<string, Group>();
  const outside: OrgMember[] = [];
  for (const m of members) {
    if (m.targets.length === 0) outside.push(m);
    for (const t of m.targets) {
      let g = byTarget.get(t.id);
      if (!g) {
        g = { id: t.id, name: label(t.name, t.id), members: [] };
        byTarget.set(t.id, g);
      }
      g.members.push({ id: m.member, name: label(m.name, m.member) });
    }
  }
  const out = [...byTarget.values()].sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
  for (const g of out) g.members.sort((a, b) => a.name.localeCompare(b.name) || a.id.localeCompare(b.id));
  return { groups: out, outside };
}

export function newer(active: number, pinnedAt: number): boolean {
  return active > pinnedAt;
}

export function validDay(raw: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(raw)) return false;
  const d = new Date(raw + "T00:00:00Z");
  return !Number.isNaN(d.getTime()) && d.toISOString().slice(0, 10) === raw;
}

export function todayUTC(now: Date): string {
  return now.toISOString().slice(0, 10);
}
