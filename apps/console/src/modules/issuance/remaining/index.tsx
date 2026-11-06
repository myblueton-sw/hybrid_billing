import type { Props } from "./contracts";
import Invoices from "./Invoices";
import Submissions from "./Submissions";
import Handoffs from "./Handoffs";
import Disclosure from "./Disclosure";
import Reports from "./Reports";
import Audit from "./Audit";
import Metering from "./Metering";
import "./screen.css";
export { financeIds } from "./contracts";
export type { Props } from "./contracts";
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
