import type { Profile } from "./api";
export type UploadIntent = {
  file: File;
  version: string;
  revision: string;
  profile: Profile;
};
export function receptionScope(profile: Profile) {
  return JSON.stringify([
    profile.profile,
    profile.workspace,
    profile.connection,
    profile.account,
    profile.period_start,
    profile.period_end,
    [...profile.versions].sort(),
    profile.max_bytes,
  ]);
}
export function validIntent(
  file: File,
  version: string,
  revision: string,
  profile: Profile,
) {
  return (
    file.size > 0 &&
    file.size <= profile.max_bytes &&
    profile.versions.includes(version) &&
    /^[1-9][0-9]{0,18}$/.test(revision) &&
    BigInt(revision) < 2n ** 63n
  );
}
