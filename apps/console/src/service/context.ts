import { createContext, useContext, useMemo } from "react";
import { adminClient, type AdminClient, type StepUp } from "./admin";
import type { Session } from "./api";

export type ServiceSession = { session: Session; signOut: () => void; stepUp?: StepUp; client?: AdminClient };

export const ServiceContext = createContext<ServiceSession | null>(null);

export function useService(): ServiceSession | null {
  return useContext(ServiceContext);
}

export function clientKey(service: ServiceSession | null): [string, StepUp | undefined] {
  return [service?.session.csrf ?? "", service?.stepUp];
}

let last: { csrf: string; stepUp: StepUp | undefined; client: AdminClient } | null = null;

export function clientFor(csrf: string, stepUp: StepUp | undefined): AdminClient {
  if (!last || last.csrf !== csrf || last.stepUp !== stepUp) last = { csrf, stepUp, client: adminClient(csrf, undefined, stepUp) };
  return last.client;
}

export function useAdminClient(): AdminClient {
  const service = useContext(ServiceContext);
  const [csrf, stepUp] = clientKey(service);
  return useMemo(() => service?.client ?? clientFor(csrf, stepUp), [service?.client, csrf, stepUp]);
}
