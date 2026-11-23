import type { SetupContext } from "./context";
import { SetupError } from "./context";
import type { SetupOptions, setupReadClient } from "./client";
import type { ConfigurationPage } from "./history";

export type ReadState<T> = Readonly<{ loading: boolean; value?: T; error?: string }>;
export type SetupReadState = Readonly<{ historyAfter: string; context: ReadState<SetupContext>; options: ReadState<SetupOptions>; history: ReadState<ConfigurationPage> }>;
type Client = Pick<ReturnType<typeof setupReadClient>, "context" | "options" | "configurations">;
const empty = (): SetupReadState => ({ historyAfter: "", context: { loading: false }, options: { loading: false }, history: { loading: false } });

export class SetupReadStore {
  private state = empty();
  private listeners = new Set<() => void>();
  private sourceGeneration = 0;
  private historyGeneration = 0;
  constructor(private readonly client: Client, readonly contract: string, readonly on: string) {}
  getSnapshot = () => this.state;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  private update(patch: Partial<SetupReadState>) { this.state = { ...this.state, ...patch }; this.listeners.forEach(listener => listener()); }
  private error(error: unknown) { return error instanceof SetupError ? error.code : "service_unavailable"; }
  dispose() { this.sourceGeneration++; this.historyGeneration++; this.state = empty(); }
  async load() { await Promise.all([this.source(), this.history()]); }
  async source() {
    const generation = ++this.sourceGeneration;
    this.update({ context: { loading: true }, options: { loading: false } });
    let context: SetupContext;
    try { context = await this.client.context(this.contract, this.on); }
    catch (error) { if (generation === this.sourceGeneration) this.update({ context: { loading: false, error: this.error(error) } }); return; }
    if (generation !== this.sourceGeneration) return;
    this.update({ context: { loading: false, value: context }, options: { loading: true } });
    try {
      const options = await this.client.options(context);
      if (generation === this.sourceGeneration) this.update({ options: { loading: false, value: options } });
    } catch (error) { if (generation === this.sourceGeneration) this.update({ options: { loading: false, error: this.error(error) } }); }
  }
  async history(next = false) {
    const previous = this.state.history.value;
    if (next && !previous?.next) return;
    const generation = ++this.historyGeneration;
    this.update({ historyAfter: next ? previous!.next : "", history: { loading: true } });
    try {
      const page = await this.client.configurations(this.contract, next ? previous!.next : "", next ? previous!.snapshot : "");
      if (generation === this.historyGeneration) this.update({ history: { loading: false, value: page } });
    } catch (error) { if (generation === this.historyGeneration) this.update({ history: { loading: false, error: this.error(error) } }); }
  }
}
