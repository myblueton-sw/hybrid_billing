import { organizations } from "./model";
export type OrganizationNode = { id: string; parent: string; depth: number };
export const enterpriseNodes: OrganizationNode[] = Array.from(
  { length: 12 },
  (_, c) => {
    const company = `C${String(c + 1).padStart(2, "0")}`;
    return [
      { id: company, parent: "", depth: 1 },
      ...Array.from({ length: 10 }, (_, d) => {
        const division = `${company}.B${String(d + 1).padStart(2, "0")}`;
        return [
          { id: division, parent: company, depth: 2 },
          ...Array.from({ length: 20 }, (_, n) => ({
            id: `${division}.D${String(n + 1).padStart(2, "0")}`,
            parent: division,
            depth: 3,
          })),
        ];
      }).flat(),
    ];
  },
).flat();
export const enterpriseLeaves = enterpriseNodes
  .filter((n) => n.depth === 3)
  .map((n) => n.id);
export function resolveOrganizations(
  enterprise: boolean,
  selection: string | null,
): string[] {
  const leaves: readonly string[] = enterprise
    ? enterpriseLeaves
    : organizations;
  if (selection === null || selection === "all") return [...leaves];
  if (enterprise && selection.startsWith("v1:")) {
    const bits = selection.slice(3);
    if (!/^[0-9a-f]{600}$/.test(bits)) return [];
    return enterpriseLeaves.filter(
      (_, i) => (parseInt(bits[Math.floor(i / 4)], 16) & (1 << (i % 4))) !== 0,
    );
  }
  const tokens = new Set(selection.split("|").filter(Boolean));
  return leaves.filter(
    (id) =>
      tokens.has(id) ||
      (enterprise &&
        (tokens.has(id.split(".")[0]) ||
          tokens.has(id.split(".").slice(0, 2).join(".")))),
  );
}
export function organizationResults(query: string, parent: string) {
  const q = query.trim().toLowerCase();
  return enterpriseNodes.filter((n) =>
    q
      ? n.id.toLowerCase().includes(q) &&
        (!parent || n.id.startsWith(parent + "."))
      : n.parent === parent,
  );
}
export function encodeOrganizations(ids: string[]) {
  const selected = new Set(ids);
  if (enterpriseLeaves.every((id) => selected.has(id))) return "all";
  if (!selected.size) return "";
  return (
    "v1:" +
    Array.from({ length: enterpriseLeaves.length / 4 }, (_, i) =>
      enterpriseLeaves
        .slice(i * 4, i * 4 + 4)
        .reduce((n, id, b) => n + (selected.has(id) ? 1 << b : 0), 0)
        .toString(16),
    ).join("")
  );
}
