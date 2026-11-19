let deliberate = false;
const releases = new Set<() => void>();

export function leaveDeliberately(): void {
  deliberate = true;
  for (const release of releases) release();
}

export function onDeliberateLeave(release: () => void): () => void {
  releases.add(release);
  return () => releases.delete(release);
}

export function leaving(): boolean {
  return deliberate;
}

export function stayAgain(): void {
  deliberate = false;
}

export function guardUnload(dirty: () => boolean): (e: BeforeUnloadEvent) => void {
  return (e) => {
    if (deliberate || !dirty()) return;
    e.preventDefault();
    e.returnValue = "";
  };
}

export function watchRestore(target: Pick<Window, "addEventListener" | "removeEventListener">): () => void {
  const restored = (e: Event) => {
    if ((e as PageTransitionEvent).persisted) deliberate = false;
  };
  target.addEventListener("pageshow", restored);
  return () => target.removeEventListener("pageshow", restored);
}
