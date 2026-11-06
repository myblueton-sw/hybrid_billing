import { fieldsByScreen } from "../../shared/domain-screen-ui/operations/contracts";
import { useEffect } from "react";
import type { Props } from "../../shared/domain-screen-ui/operations/types";
import Installation from "../../modules/administration/pages/installation/Screen";
import Integrations from "../../modules/administration/pages/integrations/Screen";
import Recovery from "../../modules/administration/pages/recovery/Screen";
import Retention from "../../modules/administration/pages/retention/Screen";
import ReleaseEvidence from "../../modules/administration/pages/release-evidence/Screen";
import Demo from "../../modules/administration/pages/demo/Screen";
import Licenses from "../../modules/administration/pages/licenses/Screen";
import Portal from "../../modules/customer-portal/pages/workspace/Screen";
import "../../shared/domain-screen-ui/operations/screen.css";
export default function Operations(p: Props) {
  const context = JSON.stringify([
    p.screenId,
    p.view,
    p.locale,
    p.params.get("scope"),
    p.params.get("customer"),
    p.params.get("period"),
    p.params.get("le"),
    p.params.get("ccy"),
    p.params.get("contract"),
    p.params.get("asof"),
  ]);
  useEffect(
    () =>
      p.onState(
        JSON.stringify({
          screen: p.screenId,
          view: p.view,
          source: "unconnected",
        }),
        [
          `${p.screenId}:data:${p.view}`,
          `${p.screenId}:state:unconnected`,
          ...(fieldsByScreen[p.screenId] ?? []).map(
            (f) => `${p.screenId}:field:${f}`,
          ),
        ],
      ),
    [p.screenId, p.view, p.onState],
  );
  const Screen =
    p.screenId === "P01"
      ? Installation
      : p.screenId === "P02"
        ? Integrations
        : p.screenId === "P03"
          ? Recovery
          : p.screenId === "P04"
            ? Retention
            : p.screenId === "P05"
              ? ReleaseEvidence
              : p.screenId === "P06"
                ? Demo
                : p.screenId === "P07"
                  ? Licenses
                  : Portal;
  return (
    <div className="ops-screen" data-screen-id={p.screenId} key={context}>
      <Screen {...p} />
    </div>
  );
}
