export type Receipt = {
  release_id: string;
  source_revision: string;
  state: string;
  operation_state: string;
  row_count: number;
  version: string;
  file_digest: string;
  manifest_digest: string;
  created_at: string;
};
export type Receipts = {
  releases: Receipt[];
  more_releases: boolean;
  quarantine: {
    receipt_id: string;
    source_revision: string;
    reason: string;
    operation_state: string;
  }[];
  more_quarantine: boolean;
};
export type Profile = {
  profile: string;
  workspace: string;
  connection: string;
  account: string;
  period_start: string;
  period_end: string;
  versions: string[];
  max_bytes: number;
};
export class ServiceError extends Error {
  constructor(
    public code: string,
    public requestId: string,
  ) {
    super(code);
  }
}
export async function request<T>(
  path: string,
  signal: AbortSignal,
  init: RequestInit = {},
  base = "/api/local-imports",
): Promise<T> {
  let response: Response;
  try {
    response = await fetch(base + path, {
      ...init,
      signal,
      cache: "no-store",
      headers: { "X-HB-Local-Client": "1", ...init.headers },
    });
  } catch (e) {
    if (signal.aborted) throw e;
    throw new ServiceError("service_unavailable", "");
  }
  let body;
  try {
    body = await response.json();
  } catch {
    throw new ServiceError("service_unavailable", "");
  }
  if (!response.ok)
    throw new ServiceError(
      body.error ?? "service_unavailable",
      body.request_id ?? "",
    );
  delete body.request_id;
  return body as T;
}
