import { createHash } from "node:crypto";

import type { BrowserContext } from "playwright";

import type {
  CapabilityManifest,
  SyncRequest,
  SyncResult,
} from "../../contracts/types.js";
import { fail, strictObject } from "../../contracts/validation.js";

export const alfaManifest: CapabilityManifest = {
  provider: "alfa",
  contractVersion: "10",
  actions: ["read_accounts", "read_transactions", "read_history"],
  products: ["current"],
  logs: [
    {
      product: "current",
      namespace: "alfa-current-v1",
      recordKinds: ["account", "transaction"],
    },
  ],
  history: { paginated: true },
};

interface Cursor {
  version: 1;
  inventory: string;
  account: number;
  page: number;
  previous?: string;
}
interface ResponseData {
  status: number;
  challenge: string | null;
  body: string;
}

export async function readAlfa(
  context: BrowserContext,
  origin: string,
  request: SyncRequest,
  headers: Record<string, string>,
  timeout: number,
  beforeProviderIO: () => void,
): Promise<SyncResult> {
  let permitted: { url: string; method: string; body?: string } | undefined;
  let violation = false;
  const page = await context.newPage();
  const entry = origin + "/history/";
  await context.route("**/*", async (route) => {
    const r = route.request();
    // The document is local. No provider script, navigation or redirect is executed.
    if (r.url() === entry && r.method() === "GET" && r.isNavigationRequest()) {
      await route.fulfill({
        status: 200,
        contentType: "text/html",
        body: "<!doctype html><title>Read context</title>",
      });
    } else if (
      permitted !== undefined &&
      r.url() === permitted.url &&
      r.method() === permitted.method &&
      (r.postData() ?? undefined) === permitted.body &&
      !r.isNavigationRequest()
    ) {
      await route.continue();
    } else {
      violation = true;
      await route.abort("blockedbyclient");
    }
  });
  await context.routeWebSocket("**/*", (socket) => {
    violation = true;
    socket.close();
  });
  context.on("page", (opened) => {
    if (opened !== page) {
      violation = true;
      void opened.close();
    }
  });
  page.on("download", (download) => {
    violation = true;
    void download.cancel();
  });
  await page.goto(entry, { timeout, waitUntil: "domcontentloaded" });
  const read = async (path: string, body?: string): Promise<ResponseData> => {
    permitted = {
      url: origin + path,
      method: body === undefined ? "GET" : "POST",
      ...(body === undefined ? {} : { body }),
    };
    beforeProviderIO();
    const response = await page.evaluate(
      async ({ url, method, body, headers, timeout }) => {
        const response = await fetch(url, {
          method,
          ...(body === undefined ? {} : { body }),
          headers: {
            ...headers,
            accept: "application/json",
            "content-type": "application/json",
          },
          credentials: "include",
          redirect: "error",
          signal: AbortSignal.timeout(timeout),
        });
        if (response.body === null) throw new Error("provider_body_missing");
        const reader = response.body.getReader();
        const chunks: Uint8Array[] = [];
        let size = 0;
        while (true) {
          const item = await reader.read();
          if (item.done) break;
          size += item.value.byteLength;
          if (size > 1024 * 1024) {
            await reader.cancel();
            throw new Error("provider_body_too_large");
          }
          chunks.push(item.value);
        }
        const bytes = new Uint8Array(size);
        let offset = 0;
        for (const chunk of chunks) {
          bytes.set(chunk, offset);
          offset += chunk.length;
        }
        return {
          status: response.status,
          challenge: response.headers.get("x-want-keep-challenge"),
          body: new TextDecoder("utf-8", { fatal: true }).decode(bytes),
        };
      },
      { ...permitted, headers, timeout },
    );
    permitted = undefined;
    if (violation || page.url() !== entry)
      throw new Error("network_policy_violation");
    if (response.status !== 200 || response.challenge !== null)
      throw new ProviderResponse(response);
    return response;
  };
  try {
    const accounts = (await read("/api/v1/account/")).body;
    const inventory = currentAccounts(accounts);
    const digest = createHash("sha256")
      .update(JSON.stringify(inventory))
      .digest("hex");
    let cursor: Cursor = { version: 1, inventory: digest, account: 0, page: 1 };
    if (request.cursor !== undefined) {
      const raw = strictObject(JSON.parse(request.cursor) as unknown, [
        "version",
        "inventory",
        "account",
        "page",
        "previous",
      ]);
      if (
        raw.version !== 1 ||
        raw.inventory !== digest ||
        !Number.isInteger(raw.account) ||
        !Number.isInteger(raw.page) ||
        (raw.account as number) < 0 ||
        (raw.account as number) >= inventory.length ||
        (raw.page as number) < 1 ||
        (raw.page as number) > 10_000 ||
        (raw.previous !== undefined &&
          (typeof raw.previous !== "string" ||
            !/^[0-9a-f]{64}$/.test(raw.previous)))
      )
        fail();
      cursor = raw as unknown as Cursor;
    }
    const cards = (await read("/api/v1/cards/masked-cards")).body;
    let operations = '{"operations":[],"pagesInfo":{"page":1,"size":20}}';
    let account: string | undefined;
    let empty = true;
    let previous: string | undefined;
    if (inventory.length > 0) {
      account = inventory[cursor.account];
      operations = (
        await read(
          "/api/v1/operations-history/operations",
          JSON.stringify({
            size: 20,
            page: cursor.page,
            filters: [{ type: "accounts", values: [account] }],
            forced: true,
          }),
        )
      ).body;
      const history = JSON.parse(operations) as Record<string, unknown>;
      const info = history.pagesInfo as Record<string, unknown> | undefined;
      if (
        !Array.isArray(history.operations) ||
        history.operations.length > 20 ||
        info?.page !== cursor.page ||
        info.size !== 20
      )
        fail();
      empty = history.operations.length === 0;
      previous = createHash("sha256")
        .update(
          JSON.stringify(
            history.operations.map(
              (op: unknown) => (op as { id?: unknown }).id,
            ),
          ),
        )
        .digest("hex");
      if (!empty && previous === cursor.previous) fail();
    }
    const loyalty = (await read("/api/v1/loyalty/accounts")).body;
    const loyaltyRoot = JSON.parse(loyalty) as Record<string, unknown>;
    if (!Array.isArray(loyaltyRoot.accounts) || loyaltyRoot.accounts.length > 8)
      fail();
    const summaries: string[] = [];
    for (const value of loyaltyRoot.accounts) {
      const id = (value as { account?: { id?: unknown } })?.account?.id;
      if (typeof id !== "string" || !/^[A-Za-z0-9_-]{1,128}$/.test(id)) fail();
      summaries.push((await read("/api/v1/loyalty/accounts/" + id)).body);
    }
    const complete =
      inventory.length === 0 ||
      (empty && cursor.account + 1 === inventory.length);
    if (!complete && !empty && cursor.page === 10_000) fail();
    const next = complete
      ? undefined
      : JSON.stringify({
          version: 1,
          inventory: digest,
          account: empty ? cursor.account + 1 : cursor.account,
          page: empty ? 1 : cursor.page + 1,
          ...(empty ? {} : { previous }),
        });
    const data = Buffer.from(
      JSON.stringify({
        version: 1,
        accounts,
        cards,
        operations,
        loyalty,
        summaries,
        account,
      }),
      "utf8",
    );
    if (data.length > 8 * 1024 * 1024) fail();
    return {
      outcome: "page",
      page: {
        ...token(request),
        ...(next === undefined ? {} : { nextCursor: next }),
        complete,
        coverage: {
          state: "partial",
          gaps: [
            "source_partial",
            "financial_mapping_unverified",
            "cashback_payout_unverified",
          ],
        },
        evidence: [
          {
            id: "alfa-page",
            mediaType: "application/json",
            data: data.toString("base64"),
            sha256: createHash("sha256").update(data).digest("hex"),
            locator: "alfa:structured-read:v1",
          },
        ],
        records: [],
      },
    };
  } catch (error) {
    if (!(error instanceof ProviderResponse)) throw error;
    const r = error.response;
    const kind =
      r.challenge === "mfa"
        ? "mfa_required"
        : r.challenge === "captcha"
          ? "captcha_required"
          : r.status === 401 || r.status === 403
            ? "reauthentication_required"
            : r.status === 429
              ? "rate_limited"
              : r.status >= 500
                ? "temporary_failure"
                : "contract_violation";
    const retryable = kind === "rate_limited" || kind === "temporary_failure";
    const data = Buffer.from(JSON.stringify({ code: kind, status: r.status }));
    return {
      outcome: "failure",
      failure: {
        ...token(request),
        kind,
        retryable,
        ...(retryable ? { retryAfterSeconds: 60 } : {}),
        safeMessage: "Provider read requires attention",
        evidence: [
          {
            id: "alfa-failure",
            mediaType: "application/json",
            data: data.toString("base64"),
            sha256: createHash("sha256").update(data).digest("hex"),
            locator: "alfa:read-status",
          },
        ],
      },
    };
  }
}

function currentAccounts(text: string): string[] {
  const root = JSON.parse(text) as Record<string, unknown>;
  if (!Array.isArray(root.accounts) || root.accounts.length > 100) fail();
  const ids: string[] = [];
  for (const value of root.accounts) {
    const a = value as {
      number?: unknown;
      type?: unknown;
      properties?: { creditCardAccount?: unknown };
    };
    if (a.type !== "EE" || a.properties?.creditCardAccount !== false) continue;
    if (
      typeof a.number !== "string" ||
      !/^[A-Za-z0-9_-]{1,128}$/.test(a.number) ||
      ids.includes(a.number)
    )
      fail();
    ids.push(a.number);
  }
  return ids.sort();
}
function token(r: SyncRequest) {
  return {
    jobId: r.jobId,
    attempt: r.attempt,
    leaseToken: r.leaseToken,
    connectionGeneration: r.connectionGeneration,
    binding: r.binding,
    admissionRevision: r.admissionRevision,
    cursor: r.cursor ?? "",
  };
}
class ProviderResponse extends Error {
  readonly response: ResponseData;
  constructor(response: ResponseData) {
    super("provider_read_failed");
    this.response = response;
  }
}
