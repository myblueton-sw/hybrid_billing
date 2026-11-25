import { SetupError } from "./context";
import type { ReadState } from "./read-store";

export class PartReadStore<T> {
  private state: ReadState<T> = { loading: false };
  private listeners = new Set<() => void>();
  private generation = 0;
  constructor(private readonly read: () => Promise<T>) {}
  getSnapshot = () => this.state;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  private update(next: ReadState<T>) { this.state = next; this.listeners.forEach(listener => listener()); }
  dispose() { this.generation++; this.state = { loading: false }; }
  async load() {
    const generation = ++this.generation;
    this.update({ loading: true });
    try {
      const value = await this.read();
      if (generation === this.generation) this.update({ loading: false, value });
    } catch (error) {
      if (generation === this.generation) this.update({ loading: false, error: error instanceof SetupError ? error.code : "service_unavailable" });
    }
  }
}
