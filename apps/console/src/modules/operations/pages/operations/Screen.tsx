import type { ScreenProps } from "../../../../shared/types";
import WorkQueue from "../../WorkQueue";
export default function Screen(props: ScreenProps) {
  return <WorkQueue {...props} screenId="operations" />;
}
