import type { Locale } from "../../../../shared/types";
import type { AccessSetting } from "../../../../service/admin";
import { s } from "./request-copy";
import { settingName, settingSource, settingValue } from "./request-model";
import { snakeBreakable } from "./who";

export function SettingsDisclosure({ l, settings }: { l: Locale; settings: AccessSetting[] }) {
  if (settings.length === 0) return null;
  return (
    <details className="live-ended access-settings">
      <summary>{s.title(l)}</summary>
      <p className="live-note">{s.lead(l)}</p>
      <div className="security-table-scroll live-table">
        <table role="table">
          <thead role="rowgroup"><tr role="row">
            <th scope="col" role="columnheader">{s.colSetting(l)}</th>
            <th scope="col" role="columnheader">{s.colValue(l)}</th>
            <th scope="col" role="columnheader">{s.colRange(l)}</th>
            <th scope="col" role="columnheader">{s.colSource(l)}</th>
          </tr></thead>
          <tbody role="rowgroup">
            {settings.map((x) => (
              <tr key={x.name} role="row">
                <td role="cell" className="live-user">{settingName(l, x)}<span className="live-sub live-mono access-id"><bdi dir="ltr">{snakeBreakable(x.name)}</bdi></span></td>
                <td role="cell" data-label={s.colValue(l)} className={x.source === "start_flag" ? "live-strong" : ""}>{settingValue(l, x)}</td>
                <td role="cell" data-label={s.colRange(l)}>{s.range(l, x.min, x.max, x.default)}</td>
                <td role="cell" data-label={s.colSource(l)}>{settingSource(l, x)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </details>
  );
}
