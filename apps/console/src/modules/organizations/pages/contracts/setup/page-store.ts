import { SetupError } from "./context";
import type { ReadState } from "./read-store";

type Paged = Readonly<{ next: string; snapshot: string }>;
export type PageState<T> = ReadState<T> & Readonly<{ after: string }>;

export class PageReadStore<T extends Paged> {
  private state: PageState<T> = { loading: false, after: "" };
  private listeners = new Set<() => void>();
  private generation = 0;
  constructor(private readonly read: (after: string, snapshot: string) => Promise<T>) {}
  getSnapshot = () => this.state;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  private update(next: PageState<T>) { this.state = next; this.listeners.forEach(listener => listener()); }
  dispose() { this.generation++; this.state = { loading: false, after: "" }; }
  async load(next = false) {
    const previous = this.state.value;
    if (next && !previous?.next) return;
    const generation = ++this.generation, after = next ? previous!.next : "";
    this.update({ loading: true, after });
    try {
      const value = await this.read(after, next ? previous!.snapshot : "");
      if (generation === this.generation) this.update({ loading: false, after, value });
    } catch (error) {
      if (generation === this.generation) this.update({ loading: false, after, error: error instanceof SetupError ? error.code : "service_unavailable" });
    }
  }
}
