const headings = "h1, h2, h3, h4, h5, h6, [role=heading]";

type Heading = { getClientRects(): { length: number }; closest(selector: string): unknown };
type Container<T> = { isConnected: boolean; tagName: string; querySelectorAll(selector: string): Iterable<T> };

export function keeperTarget<T extends Heading>(chain: Container<T>[]): T | null {
  if (!chain.some((node) => node.isConnected && (node.tagName === "MAIN" || node.tagName === "DIALOG"))) return null;
  for (const node of chain) {
    if (!node.isConnected) continue;
    for (const heading of node.querySelectorAll(headings)) {
      if (heading.getClientRects().length && !heading.closest("[aria-hidden=true], [inert]")) return heading;
    }
    if (node.tagName === "MAIN" || node.tagName === "DIALOG") return null;
  }
  return null;
}

function ancestors(start: Element): Element[] {
  const chain: Element[] = [];
  for (let node = start.parentElement; node && node.tagName !== "BODY"; node = node.parentElement) chain.push(node);
  return chain;
}

function unusable(element: HTMLElement): boolean {
  if (!element.isConnected || element.matches(":disabled") || element.closest("[inert]")) return true;
  return !element.getClientRects().length || getComputedStyle(element).visibility !== "visible";
}

export function installFocusKeeper(doc: Document): () => void {
  let last: HTMLElement | null = null;
  let chain: Element[] = [];
  let pending = false;
  let dirty = false;
  const remember = (event: FocusEvent) => {
    if (!(event.target instanceof HTMLElement) || event.target === doc.body) return;
    last = event.target;
    chain = ancestors(event.target);
  };
  const forget = () => {
    last = null;
    chain = [];
  };
  const rescue = () => {
    const was = last;
    if (!was || (doc.activeElement && doc.activeElement !== doc.body) || !unusable(was)) return;
    const target = keeperTarget<HTMLElement>((was.isConnected ? ancestors(was) : chain).map((node) => ({ isConnected: node.isConnected, tagName: node.tagName, querySelectorAll: (selector: string) => [...node.querySelectorAll<HTMLElement>(selector)].filter((heading) => getComputedStyle(heading).visibility === "visible") })));
    if (!target) return;
    if (!target.hasAttribute("tabindex")) target.setAttribute("tabindex", "-1");
    target.focus({ preventScroll: true });
  };
  const schedule = () => {
    if (pending) {
      dirty = true;
      return;
    }
    pending = true;
    dirty = false;
    requestAnimationFrame(() => setTimeout(() => {
      rescue();
      setTimeout(() => {
        pending = false;
        rescue();
        if (dirty) schedule();
      }, 120);
    }, 0));
  };
  const near = (change: MutationRecord) => {
    const was = last;
    if (!was) return false;
    if (change.type === "childList") return !was.isConnected || change.target.contains(was);
    return change.target === was || change.target.contains(was);
  };
  const observer = new MutationObserver((changes) => {
    if (!last || (doc.activeElement !== doc.body && doc.activeElement !== last)) return;
    if (changes.some(near)) schedule();
  });
  observer.observe(doc.body, { subtree: true, childList: true, attributes: true, attributeFilter: ["disabled", "hidden", "inert", "class", "style"] });
  doc.addEventListener("focusin", remember);
  doc.addEventListener("pointerdown", forget, true);
  return () => {
    observer.disconnect();
    doc.removeEventListener("focusin", remember);
    doc.removeEventListener("pointerdown", forget, true);
  };
}
