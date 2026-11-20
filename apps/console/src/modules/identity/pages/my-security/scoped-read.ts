import { useLayoutEffect, useMemo, useRef } from "react";

export type ScopedValue<T> = { scope: object; value: T };
type ReadToken = { key: unknown; active: boolean; version: number };

export function useScopedRead(key: unknown) {
  const holder = useRef<ReadToken | null>(null);
  if (!holder.current || holder.current.key !== key) {
    if (holder.current) holder.current.active = false;
    holder.current = { key,active:true,version:0 };
  }
  const token = holder.current;
  useLayoutEffect(() => {
    token.active = true;
    return () => { token.active = false;token.version++; };
  },[token]);
  return useMemo(() => ({ token,begin() {
    const version = ++token.version;
    return () => holder.current === token && token.active && token.version === version;
  } }),[token]);
}
