export type ProxyTarget = { target: string; changeOrigin: boolean };

const goApi = "http://127.0.0.1:8820";

export const readerPaths = [
  "/api/focus-overview",
  "/api/focus-commitments",
  "/api/focus-marketplace",
  "/api/focus-receipts",
  "/api/focus-connections",
  "/api/focus-collection-status",
];

const ungated: Record<string, ProxyTarget> = {
  "/api/v1": { target: goApi, changeOrigin: true },
  "/api/focus-collection-status": { target: "http://127.0.0.1:8810", changeOrigin: true },
  "/api/focus-connections": { target: "http://127.0.0.1:8810", changeOrigin: true },
  "/api/focus-receipts": { target: "http://127.0.0.1:8810", changeOrigin: true },
  "/api/credit-case-review": { target: "http://127.0.0.1:8806", changeOrigin: true },
  "/api/claim-history": { target: "http://127.0.0.1:8806", changeOrigin: true },
  "/api/approval-review": { target: "http://127.0.0.1:8806", changeOrigin: true },
  "/api/focus-marketplace": { target: "http://127.0.0.1:8806", changeOrigin: true },
  "/api/focus-commitments": { target: "http://127.0.0.1:8806", changeOrigin: true },
  "/api/focus-overview": { target: "http://127.0.0.1:8806", changeOrigin: true },
};

export function proxyFor(gated: boolean): Record<string, ProxyTarget> {
  if (!gated) return { ...ungated };
  return Object.fromEntries(Object.keys(ungated).map((p) => [p, { target: goApi, changeOrigin: true }]));
}
