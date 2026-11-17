import { k } from "./live-copy";
import { ltr } from "./live-format";
import type { Shared } from "./LiveCommand";

export function RestoreView({ l, page }: Shared) {
  const state = page.restore.state;
  if (state === "not_connected") return <p className="live-note-rule p04-lead">{k.restoreNotConnected(l)}</p>;
  return <p className="live-empty">{state ? <>{k.unknown(l)} ({ltr(state)})</> : k.unknown(l)}</p>;
}
