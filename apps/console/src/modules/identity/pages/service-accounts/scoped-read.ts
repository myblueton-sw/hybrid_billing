import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import type { Recovery } from "./recovery-context";

export const readIdentity = (...parts: (string | number)[]) =>
  JSON.stringify(parts);

export type ReadState<T> =
  | { state: "loading" }
  | { state: "error"; error: unknown }
  | { state: "ready"; value: T };
export function useScopedRead<T>(
  recovery: Recovery,
  key: string,
  load: () => Promise<T>,
  enabled = true,
) {
  const [generation, setGeneration] = useState(0);
  const token = useMemo(
    () => ({ active: false }),
    [recovery, key, generation, enabled],
  );
  const loader = useRef(load);
  useLayoutEffect(() => {
    loader.current = load;
  });
  useLayoutEffect(() => {
    token.active = true;
    return () => {
      token.active = false;
    };
  }, [token]);
  const [result, setResult] = useState<{
    token: typeof token;
    state: ReadState<T>;
  } | null>(null);
  useEffect(() => {
    if (!enabled) return;
    let active = true;
    const current = () => active && token.active && recovery.issues.current();
    Promise.resolve()
      .then(() => {
        if (!current()) return;
        return loader.current();
      })
      .then(
        (value) => {
          if (current() && value !== undefined)
            setResult({ token, state: { state: "ready", value } });
        },
        (error: unknown) => {
          if (current()) setResult({ token, state: { state: "error", error } });
        },
      );
    return () => {
      active = false;
    };
  }, [token, recovery, enabled]);
  return {
    result: (enabled &&
    token.active &&
    recovery.issues.current() &&
    result?.token === token
      ? result.state
      : { state: "loading" }) as ReadState<T>,
    reload: () => setGeneration((n) => n + 1),
  };
}
