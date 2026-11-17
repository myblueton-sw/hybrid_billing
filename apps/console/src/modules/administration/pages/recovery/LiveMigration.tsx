import { k } from "./live-copy";
import { Code, ListOr } from "./live-format";
import type { Shared } from "./LiveCommand";

export function MigrationView({ l, page }: Shared) {
  const m = page.migration;
  return (
    <section className="p03-unconnected" aria-labelledby="p03-later-title">
      <h3 id="p03-later-title">{k.laterTitle(l)}</h3>
      <ListOr l={l} items={m.modules}>
        {(list) => (
          <ul className="p03-effect-list">
            {list.map((x, n) => <li key={x + n}><Code l={l} value={x} say={k.migrationModule} /></li>)}
          </ul>
        )}
      </ListOr>
    </section>
  );
}
