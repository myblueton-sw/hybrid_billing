import type { Profile } from "../shared/types";
export const profiles: Profile[] = [
  {
    id: "standard",
    modules: [
      "identity",
      "design-review",
      "sources",
      "operations",
      "organizations",
      "issuance",
      "onboarding",
    ],
    extensions: [],
  },
  {
    id: "evidence",
    modules: [
      "identity",
      "design-review",
      "sources",
      "operations",
      "organizations",
      "issuance",
      "onboarding",
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
