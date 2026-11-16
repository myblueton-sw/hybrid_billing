export type Session = {
  user: string;
  workspace: string;
  role: string;
  csrf: string;
};
export type LinkInfo =
  | { purpose: "set" | "reset"; login: string }
  | { purpose: "activate" };
export type Failure =
  | "unauthenticated"
  | "limited"
  | "link"
  | "password"
  | "form"
  | "expired"
  | "unavailable";
export class ServiceError extends Error {
  constructor(
    readonly failure: Failure,
    readonly retryAfter = 0,
  ) {
    super(failure);
  }
}
type Fetch = typeof fetch;
const base = "/api/v1";
function failureOf(status: number, code: string): Failure {
  if (status === 429) return "limited";
  if (code === "link_invalid") return "link";
  if (code === "password_form") return "password";
  if (status === 401) return "unauthenticated";
  if (status === 403 && code === "csrf_refused") return "expired";
  if (status === 422 || status === 400) return "form";
  return "unavailable";
}
async function send(
  f: Fetch,
  path: string,
  init: RequestInit,
): Promise<Response> {
  let r: Response;
  try {
    r = await f(base + path, {
      credentials: "same-origin",
      cache: "no-store",
      ...init,
    });
  } catch {
    throw new ServiceError("unavailable");
  }
  if (r.ok) return r;
  let code = "";
  try {
    code = String((await r.json()).code ?? "");
  } catch {
    code = "";
  }
  const retry = Number(r.headers.get("Retry-After") ?? "0");
  throw new ServiceError(
    failureOf(r.status, code),
    Number.isFinite(retry) ? retry : 0,
  );
}
const client = { "X-HB-Client": "console" };
export function serviceClient(f: Fetch = fetch.bind(globalThis)) {
  let presession: Promise<string> | null = null;
  function pre(): Promise<string> {
    if (!presession) {
      presession = send(f, "/auth/csrf", { headers: client })
        .then(async (r) => {
          const t = await r.json().catch(() => ({}));
          if (typeof t.csrf_token !== "string" || !t.csrf_token) throw new ServiceError("unavailable");
          return t.csrf_token as string;
        })
        .catch((e) => {
          presession = null;
          throw e;
        });
    }
    return presession;
  }
  async function post(path: string, body: Record<string, string>, retried = false): Promise<Response> {
    const issued = pre();
    const token = await issued;
    try {
      return await send(f, path, {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-CSRF-Token": token },
        body: JSON.stringify(body),
      });
    } catch (e) {
      if (e instanceof ServiceError && (e.failure === "unavailable" || e.failure === "expired") && presession === issued) presession = null;
      if (e instanceof ServiceError && e.failure === "expired" && !retried) return post(path, body, true);
      throw e;
    }
  }
  const session = (j: Record<string, unknown>): Session => ({
    user: String(j.user),
    workspace: String(j.workspace),
    role: String(j.role),
    csrf: String(j.csrf_token),
  });
  return {
    async current(): Promise<Session | null> {
      try {
        const r = await send(f, "/session", { headers: client });
        return session(await r.json());
      } catch (e) {
        if (e instanceof ServiceError && e.failure === "unauthenticated")
          return null;
        throw e;
      }
    },
    async login(workspace: string, login: string, password: string) {
      const r = await post("/auth/login", { workspace, login, password });
      return session(await r.json());
    },
    async logout(s: Session) {
      await send(f, "/auth/logout", {
        method: "POST",
        headers: { "X-CSRF-Token": s.csrf },
      });
    },
    async peek(token: string): Promise<LinkInfo> {
      const r = await post("/auth/link", { token });
      const j = await r.json();
      return j.purpose === "activate"
        ? { purpose: "activate" }
        : { purpose: j.purpose === "reset" ? "reset" : "set", login: String(j.login) };
    },
    async redeem(
      token: string,
      password: string,
      first?: { login: string; email: string },
    ) {
      await post("/auth/redeem", {
        token,
        password,
        ...(first ? { login: first.login, email: first.email } : {}),
      });
    },
  };
}
export type ServiceClient = ReturnType<typeof serviceClient>;
export function fragmentToken(hash: string): string {
  const t = hash.replace(/^#/, "");
  return /^[A-Za-z0-9_-]{43}$/.test(t) ? t : "";
}
