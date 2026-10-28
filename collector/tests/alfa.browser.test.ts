import { readFileSync } from "node:fs";
import { createServer } from "node:http";
import { afterEach, describe, expect, it } from "vitest";
import { alfaManifest } from "../src/providers/alfa/read.js";
import { parseRuntimeConfig } from "../src/runtime/config.js";
import {
  CollectorRuntime,
  CollectorBeforeIOUnavailableError,
} from "../src/runtime/runtime.js";
import { parseSyncResult } from "../src/contracts/ingestion.js";

const fixture = (name: string) =>
  readFileSync(
    new URL(`../contracts/alfa/fixtures/${name}.json`, import.meta.url),
    "utf8",
  );
const cleanup: Array<() => Promise<void>> = [];
afterEach(async () => {
  for (const close of cleanup.splice(0).reverse()) await close();
});
const digest = "sha256:" + "0".repeat(64);
const binding = {
  provider: "alfa",
  environment: "test",
  adapterBuildDigest: digest,
  collectorImageDigest: digest,
  contractVersion: "10",
  allowlistRevision: "alfa-read-v1",
  nonSecretConfigRevision: "test-v1",
  operatorPermissionRevision: "synthetic-v1",
} as const;
const headers = {
  "x-xsrf-token": "synthetic-xsrf",
  "x-security-request": "synthetic-security",
  "device-app-id": "synthetic-device",
  "ao-origin-application-id": "synthetic-app",
  "zone-offset": "180",
};
function request(cursor?: string) {
  return {
    jobId: "11111111-1111-4111-8111-111111111111",
    connectionId: "22222222-2222-4222-8222-222222222222",
    attempt: 1,
    leaseToken: "synthetic-lease",
    connectionGeneration: 1,
    binding,
    admissionRevision: 1,
    ...(cursor === undefined ? {} : { cursor }),
  };
}
async function setup(mode = "safe") {
  const calls: Array<{
    method: string;
    path: string;
    cookie?: string;
    body: string;
  }> = [];
  const server = createServer(async (req, res) => {
    let body = "";
    for await (const chunk of req) body += String(chunk);
    calls.push({
      method: req.method!,
      path: req.url!,
      ...(req.headers.cookie === undefined
        ? {}
        : { cookie: req.headers.cookie }),
      body,
    });
    if (mode === "redirect" && req.url === "/api/v1/account/") {
      res.writeHead(302, { location: "/payment" });
      res.end();
      return;
    }
    if (
      mode === "mfa" ||
      mode === "captcha" ||
      mode === "expired" ||
      mode === "rate"
    ) {
      res.writeHead(
        mode === "rate" ? 429 : 403,
        mode === "expired" || mode === "rate"
          ? {}
          : { "x-want-keep-challenge": mode },
      );
      res.end("synthetic-secret");
      return;
    }
    res.setHeader("content-type", "application/json");
    switch (req.url) {
      case "/api/v1/account/":
        res.end(fixture("accounts"));
        break;
      case "/api/v1/cards/masked-cards":
        res.end(fixture("cards"));
        break;
      case "/api/v1/loyalty/accounts":
        res.end(fixture("loyalty"));
        break;
      case "/api/v1/loyalty/accounts/synthetic-loyalty-a":
        res.end(fixture("summary"));
        break;
      case "/api/v1/operations-history/operations": {
        const parsed = JSON.parse(body) as { page: number };
        res.end(
          mode === "repeat"
            ? JSON.stringify({
                ...JSON.parse(fixture("history")),
                pagesInfo: { page: parsed.page, size: 20 },
              })
            : parsed.page === 1
              ? fixture("history")
              : JSON.stringify({
                  operations: [],
                  pagesInfo: { page: parsed.page, size: 20 },
                }),
        );
        break;
      }
      default:
        res.writeHead(500);
        res.end("unexpected_mutation");
    }
  });
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  cleanup.push(
    () =>
      new Promise<void>((resolve, reject) =>
        server.close((e) => (e ? reject(e) : resolve())),
      ),
  );
  const address = server.address();
  if (address === null || typeof address === "string")
    throw new Error("fixture_port_missing");
  const origin = `http://127.0.0.1:${address.port}`;
  const runtime = new CollectorRuntime(
    parseRuntimeConfig({
      version: 1,
      socket: "/tmp/synthetic-alfa.sock",
      requestTimeoutMs: 10_000,
      bindings: [
        {
          binding,
          admissionRevision: 1,
          manifest: alfaManifest,
          origin,
          routes: [
            { method: "GET", path: "/history/", action: "entry" },
            { method: "GET", path: "/api/v1/account/", action: "read" },
          ],
        },
      ],
    }),
  );
  cleanup.push(() => runtime.close());
  const read = (sid = "member-a", cursor?: string) =>
    runtime.read(
      {
        version: 1,
        syncRequest: request(cursor),
        storageState: {
          cookies: [
            {
              name: "sid",
              value: sid,
              domain: "127.0.0.1",
              path: "/",
              expires: -1,
              httpOnly: true,
              secure: false,
              sameSite: "Lax",
            },
          ],
          origins: [],
        },
        sessionHeaders: headers,
      },
      new AbortController().signal,
    );
  return { calls, read, runtime };
}

describe("Alfa browser boundary", () => {
  it("reads only closed routes, preserves raw tokens, paginates and isolates sessions", async () => {
    const f = await setup();
    const first = await f.read();
    expect(parseSyncResult(first, request())).toEqual(first);
    expect(first.page?.complete).toBe(false);
    const data = Buffer.from(
      first.page!.evidence[0]!.data,
      "base64",
    ).toString();
    expect(data).toContain("900719925474099312345");
    expect(JSON.stringify(first)).not.toContain("synthetic-xsrf");
    expect(first.page?.coverage.state).toBe("partial");
    const cursor = first.page!.nextCursor!;
    const last = await f.read("member-b", cursor);
    expect(last.page?.complete).toBe(true);
    expect(
      f.calls.every(
        (c) =>
          c.method === "GET" ||
          (c.path === "/api/v1/operations-history/operations" &&
            c.method === "POST"),
      ),
    ).toBe(true);
    expect(
      f.calls.some(
        (c) => c.path === "/history/" || c.path.includes("last-seen-balance"),
      ),
    ).toBe(false);
    const history = f.calls.filter((c) => c.method === "POST");
    expect(JSON.parse(history[0]!.body)).toEqual({
      size: 20,
      page: 1,
      filters: [{ type: "accounts", values: ["synthetic-current-a"] }],
      forced: true,
    });
    expect(history[0]?.cookie).toBe("sid=member-a");
    expect(history[1]?.cookie).toBe("sid=member-b");
    const browser = Reflect.get(f.runtime, "browser") as {
      contexts(): unknown[];
    };
    expect(browser.contexts()).toHaveLength(0);
  });
  it("rejects a repeated history page without advancing its cursor", async () => {
    const f = await setup("repeat");
    const first = await f.read();
    await expect(f.read("member-a", first.page!.nextCursor!)).rejects.toThrow();
  });
  it("keeps page-setup failures before provider IO and disposes the context", async () => {
    const f = await setup();
    let closed = false;
    Reflect.set(f.runtime, "browser", {
      isConnected: () => true,
      newContext: async () => ({
        newPage: async () => {
          throw new Error("synthetic_page_setup_failure");
        },
        close: async () => {
          closed = true;
        },
      }),
      close: async () => undefined,
    });
    await expect(f.read()).rejects.toBeInstanceOf(
      CollectorBeforeIOUnavailableError,
    );
    expect(f.calls).toHaveLength(0);
    expect(closed).toBe(true);
  });
  it("never follows redirects into payment routes", async () => {
    const f = await setup("redirect");
    await expect(f.read()).rejects.toThrow();
    expect(f.calls.map((c) => c.path)).toEqual(["/api/v1/account/"]);
  });
  it.each([
    ["mfa", "mfa_required"],
    ["captcha", "captcha_required"],
    ["expired", "reauthentication_required"],
    ["rate", "rate_limited"],
  ])("reports %s without retaining response bodies", async (mode, kind) => {
    const f = await setup(mode);
    const result = await f.read();
    expect(result.failure?.kind).toBe(kind);
    expect(JSON.stringify(result)).not.toContain("synthetic-secret");
    expect(
      Buffer.from(result.failure!.evidence[0]!.data, "base64").toString(),
    ).not.toContain("synthetic-secret");
    expect(f.calls).toHaveLength(1);
  });
});
