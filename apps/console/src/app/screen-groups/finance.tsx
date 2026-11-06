import type { Props } from "../../shared/domain-screen-ui/finance/contracts";
import Invoices from "../../modules/issuance/pages/invoices/Screen";
import Submissions from "../../modules/delivery/pages/submissions/Screen";
import Handoffs from "../../modules/delivery/pages/handoffs/Screen";
import Disclosure from "../../modules/delivery/pages/exports/Screen";
import Reports from "../../modules/analytics/pages/reports/Screen";
import Audit from "../../modules/history/pages/activity-history/Screen";
import Metering from "../../modules/sources/pages/meters/Screen";
import "../../shared/domain-screen-ui/finance/screen.css";
export { financeIds } from "../../shared/domain-screen-ui/finance/contracts";
export type { Props } from "../../shared/domain-screen-ui/finance/contracts";
const components = {
  I01: Invoices,
  I02: Submissions,
  I03: Handoffs,
  I04: Disclosure,
  R01: Reports,
  R02: Audit,
  B02: Metering,
};
export default function FinanceScreens(props: Props) {
  const Component = components[props.screenId as keyof typeof components];
  const context = JSON.stringify([
    props.screenId,
    props.locale,
    props.view,
    ...["scope", "le", "customer", "contract", "period", "ccy", "asof"].map(
      (k) => props.params.get(k),
    ),
  ]);
  return Component ? <Component key={context} {...props} /> : null;
}
