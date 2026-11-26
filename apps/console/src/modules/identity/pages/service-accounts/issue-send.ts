import type { AdminClient } from "../../../../service/admin";
import type { Exec } from "./issue-attempts";

export function issueExec(api: AdminClient, service: string): Exec {
  return (tried, expected, frozen) =>
    frozen.rotate
      ? api.command("service.rotate_credential", frozen.rotate.credential, expected, frozen.reason, { days: frozen.days, overlap_minutes: frozen.rotate.overlap }, tried)
      : api.command("service.issue_credential", service, expected, frozen.reason, { days: frozen.days }, tried);
}
