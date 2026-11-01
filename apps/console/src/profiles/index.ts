import type { Profile } from "../shared/types";
export const profiles: Profile[] = [
  {
    id: "standard",
    modules: [
      "design-review",
      "sources",
      "operations",
      "organizations",
      "issuance",
    ],
    extensions: [],
  },
  {
    id: "evidence",
    modules: [
      "design-review",
      "sources",
      "operations",
      "organizations",
      "issuance",
    ],
    extensions: ["evidence-guide"],
  },
];
export function getProfile(id: string) {
  if (new Set(profiles.map((p) => p.id)).size !== profiles.length)
    throw Error("duplicate profile");
  const profile = profiles.find((p) => p.id === id);
  if (!profile) throw Error("unknown profile");
  return profile;
}
