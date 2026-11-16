import { createContext, useContext } from "react";
import type { Session } from "./api";

export type ServiceSession = { session: Session; signOut: () => void };

export const ServiceContext = createContext<ServiceSession | null>(null);

export function useService(): ServiceSession | null {
  return useContext(ServiceContext);
}
