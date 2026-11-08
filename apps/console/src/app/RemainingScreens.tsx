import { useCallback, useRef, useState } from "react";
import type { Locale } from "../shared/types";
import type { Audience } from "../planning/routing";
import { remainingGroups } from "./remaining-screens";
import Security from "./screen-groups/identity";
import Finance from "./screen-groups/finance";
import Work from "./screen-groups/work";
import Operations from "./screen-groups/operations";
type Props = {
  screenId: string;
  locale: Locale;
  params: URLSearchParams;
  view: string;
  audience: Audience;
  onState: (state: string, items: string[]) => void;
  onDirty: (dirty: boolean) => void;
};
function RevisionScope(props: Props) {
  const [editRevision, setEditRevision] = useState(0);
  const fragments = useRef(new Map<string, string>());
  const itemIds = useRef(new Set<string>());
  const report = props.onState;
  const requestedContext = JSON.stringify({
    screen: props.screenId,
    locale: props.locale,
    view: props.view,
    audiencePresentation: props.audience,
    authority: "unverified",
    scope: Object.fromEntries(
      ["scope", "customer", "contract", "period", "le", "ccy", "asof"].map(
        (k) => [k, props.params.get(k)],
      ),
    ),
  });
  const onState = useCallback(
    (state: string, items: string[]) => {
      for (const id of items) itemIds.current.add(id);
      fragments.current.set(items.join("|"), state);
      report(
        JSON.stringify({
          editRevision,
          requestedContext: JSON.parse(requestedContext),
          fragments: [...fragments.current].sort(([a], [b]) =>
            a.localeCompare(b),
          ),
        }),
        [...itemIds.current].sort(),
      );
    },
    [report, editRevision, requestedContext],
  );
  const p = { ...props, onState };
  const leaf = (remainingGroups.security as readonly string[]).includes(
    p.screenId,
  ) ? (
    <Security {...p} />
  ) : (remainingGroups.finance as readonly string[]).includes(p.screenId) ? (
    <Finance {...p} />
  ) : (remainingGroups.work as readonly string[]).includes(p.screenId) ? (
    <Work {...p} audience={p.audience === "customer" ? "customer" : "admin"} />
  ) : (
    <Operations {...p} />
  );
  return (
    <div
      data-remaining-screen={p.screenId}
      data-remaining-view={p.view}
      onChangeCapture={() => setEditRevision((n) => n + 1)}
      onSubmitCapture={() => setEditRevision((n) => n + 1)}
      onClickCapture={(event) => {
        if (
          event.target instanceof Element &&
          event.target.closest("button,summary")
        )
          setEditRevision((n) => n + 1);
      }}
    >
      {leaf}
    </div>
  );
}
export default function RemainingScreens(props: Props) {
  const context = JSON.stringify([
    props.screenId,
    props.locale,
    props.view,
    props.audience,
    ...["scope", "customer", "contract", "period", "le", "ccy", "asof"].map(
      (k) => props.params.get(k),
    ),
  ]);
  return <RevisionScope key={context} {...props} />;
}
