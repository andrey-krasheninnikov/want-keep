import { readFileSync } from "node:fs";
import { request } from "node:http";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer, type Server } from "node:http";
import { gzipSync } from "node:zlib";

import { chromium, type BrowserContext } from "playwright";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  parseRuntimeConfig,
  type RuntimeConfig,
} from "../src/runtime/config.js";
import {
  CollectorBeforeIOUnavailableError,
  CollectorRuntime,
} from "../src/runtime/runtime.js";
import { startCollectorServer } from "../src/runtime/server.js";

const manifest = JSON.parse(
  readFileSync(
    new URL("../contracts/v10/fixtures/manifest.json", import.meta.url),
    "utf8",
  ),
) as Record<string, unknown>;
const golden = JSON.parse(
  readFileSync(
    new URL("../contracts/v10/fixtures/golden-page.json", import.meta.url),
    "utf8",
  ),
) as { page: Record<string, unknown> };
const binding = golden.page.binding as Record<string, unknown>;
const requestBody = {
  jobId: golden.page.jobId,
  attempt: golden.page.attempt,
  leaseToken: golden.page.leaseToken,
  connectionId: "22222222-2222-4222-8222-222222222222",
  connectionGeneration: golden.page.connectionGeneration,
  binding,
  admissionRevision: golden.page.admissionRevision,
};

const cleanup: Array<() => Promise<void>> = [];
afterEach(async () => {
  for (const close of cleanup.splice(0).reverse()) await close();
});

describe("collector security boundary", () => {
  it("reads an allowlisted page and statement through isolated contexts", async () => {
    const portal = await startPortal("safe");
    const collector = await start(portal.origin, "/portal");
    const first = await post(collector.socket, "/v1/read", envelope("alpha"));
    const second = await post(
      collector.socket,
      "/v1/read",
      envelope("beta", {
        from: "2026-09-01T00:00:00Z",
        to: "2026-09-02T00:00:00Z",
      }),
    );
    expect(first.status).toBe(200);
    expect(second.status).toBe(200);
    expect(first.body).not.toContain("alpha");
    expect(second.body).not.toContain("beta");
    expect(JSON.parse(first.body).page.records).toHaveLength(17);
  });

  it("restarts Chromium after a disconnected browser", async () => {
    const portal = await startPortal("safe");
    const runtime = new CollectorRuntime(config(portal.origin, "/portal"));
    cleanup.push(() => runtime.close());
    expect(
      (await runtime.read(envelope("alpha"), new AbortController().signal))
        .outcome,
    ).toBe("page");
    const browser = Reflect.get(runtime, "browser") as {
      close(): Promise<void>;
    };
    await browser.close();
    expect(
      (await runtime.read(envelope("alpha"), new AbortController().signal))
        .outcome,
    ).toBe("page");
  });

  it("reports browser launch failure before provider IO", async () => {
    const portal = await startPortal("safe");
    const collector = await start(portal.origin, "/portal");
    const launch = vi
      .spyOn(chromium, "launch")
      .mockRejectedValueOnce(new Error("synthetic launch failure"));
    try {
      const result = await post(
        collector.socket,
        "/v1/read",
        envelope("alpha"),
      );
      expect(result).toEqual({
        status: 503,
        body: '{"code":"collector_before_io_unavailable"}',
      });
      expect(portal.requests()).toBe(0);
    } finally {
      launch.mockRestore();
    }
  });

  it("keeps context setup failure before provider IO", async () => {
    const portal = await startPortal("safe");
    const runtime = new CollectorRuntime(config(portal.origin, "/portal"));
    Reflect.set(runtime, "browser", {
      isConnected: () => true,
      newContext: async () => {
        throw new Error("synthetic context failure");
      },
      close: async () => undefined,
    });
    await expect(
      runtime.read(envelope("alpha"), new AbortController().signal),
    ).rejects.toBeInstanceOf(CollectorBeforeIOUnavailableError);
    expect(portal.requests()).toBe(0);
  });

  it("cancels a read when the Unix socket closes after the complete request", async () => {
    const portal = await startPortal("safe");
    const collector = await start(portal.origin, "/portal");
    let signal: AbortSignal | undefined;
    const read = vi
      .spyOn(CollectorRuntime.prototype, "read")
      .mockImplementationOnce(async (_value, requestSignal) => {
        signal = requestSignal;
        return await new Promise((_, reject) => {
          requestSignal.addEventListener(
            "abort",
            () => reject(new Error("request_aborted")),
            { once: true },
          );
        });
      });
    const data = JSON.stringify(envelope("alpha"));
    const call = request({
      socketPath: collector.socket,
      path: "/v1/read",
      method: "POST",
      headers: {
        "content-type": "application/json",
        "content-length": Buffer.byteLength(data),
      },
    });
    call.on("error", () => undefined);
    try {
      call.end(data);
      await vi.waitFor(() => expect(read).toHaveBeenCalledOnce());
      expect(signal?.aborted).toBe(false);
      call.destroy();
      await vi.waitFor(() => expect(signal?.aborted).toBe(true));
      expect(portal.requests()).toBe(0);
    } finally {
      call.destroy();
      read.mockRestore();
    }
  });

  it("passes the issued cursor to the next read page", async () => {
    const portal = await startPortal("paged");
    const collector = await start(portal.origin, "/portal");
    const first = await post(collector.socket, "/v1/read", envelope("alpha"));
    const second = await post(
      collector.socket,
      "/v1/read",
      envelope("alpha", undefined, "page-2"),
    );
    expect(first.status).toBe(200);
    expect(JSON.parse(first.body).page.nextCursor).toBe("page-2");
    expect(second.status).toBe(200);
    expect(JSON.parse(second.body).page.cursor).toBe("page-2");
    expect(JSON.parse(second.body).page.complete).toBe(true);
  });

  it("cancels stalled context setup and closes a late context", async () => {
    const portal = await startPortal("safe");
    const runtime = new CollectorRuntime(config(portal.origin, "/portal"));
    cleanup.push(() => runtime.close());
    let finish!: (context: BrowserContext) => void;
    const newContext = vi.fn(
      () =>
        new Promise<BrowserContext>((resolve) => {
          finish = resolve;
        }),
    );
    const closeBrowser = vi.fn(async () => undefined);
    Reflect.set(runtime, "browser", {
      isConnected: () => true,
      newContext,
      close: closeBrowser,
    });
    const controller = new AbortController();
    const rejected = expect(
      runtime.read(envelope("alpha"), controller.signal),
    ).rejects.toBeInstanceOf(CollectorBeforeIOUnavailableError);
    await vi.waitFor(() => expect(newContext).toHaveBeenCalledOnce());
    controller.abort();
    await rejected;
    expect(closeBrowser).toHaveBeenCalled();
    expect(Reflect.get(runtime, "busy")).toBe(false);
    expect(Reflect.get(runtime, "browser")).toBeUndefined();
    const closeContext = vi.fn(async () => undefined);
    finish({ close: closeContext } as unknown as BrowserContext);
    await vi.waitFor(() => expect(closeContext).toHaveBeenCalledOnce());
    expect(portal.requests()).toBe(0);
    expect(
      (await runtime.read(envelope("alpha"), new AbortController().signal))
        .outcome,
    ).toBe("page");
  });

  it("cancels browser launch at the job deadline", async () => {
    const portal = await startPortal("safe");
    const runtime = new CollectorRuntime(config(portal.origin, "/portal"));
    const launch = vi.spyOn(chromium, "launch").mockImplementationOnce(
      (options) =>
        new Promise((_, reject) => {
          (Reflect.get(options!, "signal") as AbortSignal).addEventListener(
            "abort",
            () => reject(new Error("cancelled")),
            { once: true },
          );
        }),
    );
    try {
      await expect(
        runtime.read(envelope("alpha"), AbortSignal.timeout(10)),
      ).rejects.toBeInstanceOf(CollectorBeforeIOUnavailableError);
      expect(launch.mock.calls[0]![0]!.timeout).toBe(10_000);
      expect(Reflect.get(runtime, "busy")).toBe(false);
      expect(portal.requests()).toBe(0);
    } finally {
      launch.mockRestore();
    }
  });

  it.each(["oversized-entry", "compressed-oversized-entry"])(
    "bounds the decoded %s before the financial read",
    async (scenario) => {
      const portal = await startPortal(scenario);
      const collector = await start(portal.origin, "/portal");
      const result = await post(
        collector.socket,
        "/v1/read",
        envelope("alpha"),
      );
      expect(result.status).toBe(503);
      expect(portal.requests()).toBe(1);
    },
  );

  it("decodes an entry and preserves its session cookies", async () => {
    const portal = await startPortal("compressed-entry");
    const collector = await start(portal.origin, "/portal");
    const result = await post(collector.socket, "/v1/read", envelope("alpha"));
    expect(result.status).toBe(200);
    expect(JSON.parse(result.body).page.records).toHaveLength(17);
  });

  it("ignores a provider page that replaces fetch", async () => {
    const portal = await startPortal("forged-fetch");
    const collector = await start(portal.origin, "/portal");
    const result = await post(collector.socket, "/v1/read", envelope("alpha"));
    expect(result.status).toBe(200);
    expect(JSON.parse(result.body).page.records).toHaveLength(17);
  });

  it("prevents the provider page from sending a second statement request", async () => {
    const portal = await startPortal("statement-hijack");
    const collector = await start(portal.origin, "/portal");
    const result = await post(
      collector.socket,
      "/v1/read",
      envelope("alpha", {
        from: "2026-09-01T00:00:00Z",
        to: "2026-09-02T00:00:00Z",
      }),
    );
    expect(result.status).toBe(200);
    expect(portal.statementRequests()).toBe(1);
  });

  it("removes WebRTC before provider scripts run", async () => {
    const portal = await startPortal("webrtc");
    const collector = await start(portal.origin, "/portal");
    const result = await post(collector.socket, "/v1/read", envelope("alpha"));
    expect(result.status).toBe(200);
  });

  it("keeps sessions separate and maps owner challenges", async () => {
    const portal = await startPortal("session");
    const collector = await start(portal.origin, "/portal");
    const first = await post(collector.socket, "/v1/read", envelope("alpha"));
    const second = await post(collector.socket, "/v1/read", envelope("beta"));
    expect(
      JSON.parse(first.body).page.records[1].balanceSnapshot.freshness,
    ).toBe("fresh");
    expect(
      JSON.parse(second.body).page.records[1].balanceSnapshot.freshness,
    ).toBe("stale");

    for (const [scenario, kind] of [
      ["mfa", "mfa_required"],
      ["captcha", "captcha_required"],
      ["expired", "reauthentication_required"],
    ]) {
      const challengePortal = await startPortal(scenario);
      const challenged = await start(challengePortal.origin, "/portal");
      const result = await post(
        challenged.socket,
        "/v1/read",
        envelope("alpha"),
      );
      expect(JSON.parse(result.body).failure.kind).toBe(kind);
    }
  });

  it("keeps an invalid authorization body and rate limit typed", async () => {
    const invalid = await startPortal("invalid-auth-body");
    const authCollector = await start(invalid.origin, "/portal");
    const auth = await post(
      authCollector.socket,
      "/v1/read",
      envelope("alpha"),
    );
    expect(JSON.parse(auth.body).failure.kind).toBe(
      "reauthentication_required",
    );

    const limited = await startPortal("rate-limit");
    const limitCollector = await start(limited.origin, "/portal");
    const limit = await post(
      limitCollector.socket,
      "/v1/read",
      envelope("alpha"),
    );
    expect(JSON.parse(limit.body).failure).toMatchObject({
      kind: "rate_limited",
      retryable: true,
      retryAfterSeconds: 60,
    });
  });

  it.each(["payment", "popup", "websocket"])(
    "prevents %s behavior in the provider page",
    async (scenario) => {
      const portal = await startPortal(scenario);
      const collector = await start(portal.origin, "/portal");
      const result = await post(
        collector.socket,
        "/v1/read",
        envelope("alpha"),
      );
      expect(result.status).toBe(200);
      expect(portal.actionRequests()).toBe(0);
    },
  );

  it.each(["same origin", "different origin"])(
    "blocks an entry redirect to a forbidden target on %s before IO",
    async (destinationOrigin) => {
      const destination = await startPortal("safe");
      const portal = await startPortal(
        "redirect",
        destinationOrigin === "same origin"
          ? "/unknown"
          : destination.origin + "/unknown",
      );
      const collector = await start(portal.origin, "/portal");
      const result = await post(
        collector.socket,
        "/v1/read",
        envelope("alpha"),
      );
      expect(result.status).toBe(503);
      expect(portal.requests()).toBe(1);
      expect(destination.requests()).toBe(0);
    },
  );

  it("blocks the page download request", async () => {
    const portal = await startPortal("download");
    const collector = await start(portal.origin, "/portal");
    await post(collector.socket, "/v1/read", envelope("alpha"));
    expect(portal.downloadRequests()).toBe(0);
  });

  it("blocks service workers without failing the allowed read", async () => {
    const portal = await startPortal("serviceworker");
    const collector = await start(portal.origin, "/portal");
    const result = await post(collector.socket, "/v1/read", envelope("alpha"));
    expect(result.status).toBe(200);
    expect(portal.workerRequests()).toBe(0);
  });

  it("rejects mutation capabilities and stale bindings before browser work", async () => {
    const portal = await startPortal("safe");
    expect(() =>
      config(portal.origin, "/portal", [
        { method: "GET", path: "/portal", action: "entry" },
        { method: "GET", path: "/api/read", action: "read" },
        { method: "POST", path: "/api/payment", action: "payment" },
      ]),
    ).toThrow("invalid ingestion contract");
    const collector = await start(portal.origin, "/portal");
    const stale = envelope("alpha") as {
      syncRequest: { admissionRevision: number };
    };
    stale.syncRequest.admissionRevision = 2;
    const result = await post(collector.socket, "/v1/read", stale);
    expect(result.status).toBe(422);
    expect(JSON.parse(result.body).code).toBe("collector_preflight_rejected");
  });

  it("rejects route normalization and malformed sessions before provider IO", async () => {
    const portal = await startPortal("safe");
    for (const path of ["//127.0.0.1/private", "/api/../private"]) {
      expect(() =>
        config(portal.origin, "/portal", [
          { method: "GET", path: "/portal", action: "entry" },
          { method: "GET", path, action: "read" },
        ]),
      ).toThrow("invalid ingestion contract");
    }
    const collector = await start(portal.origin, "/portal");
    const malformed = envelope("alpha") as {
      storageState: { cookies: unknown[] };
    };
    malformed.storageState.cookies = [null];
    const result = await post(collector.socket, "/v1/read", malformed);
    expect(result.status).toBe(422);
    expect(JSON.parse(result.body).code).toBe("collector_session_invalid");
    expect(portal.requests()).toBe(0);
  });

  it("rejects missing statement capability and allowed-route redirects before acceptance", async () => {
    const portal = await startPortal("statement-redirect");
    const withoutStatement = await start(portal.origin, "/portal", [
      { method: "GET", path: "/portal", action: "entry" },
      { method: "GET", path: "/api/read", action: "read" },
    ]);
    const missing = await post(
      withoutStatement.socket,
      "/v1/read",
      envelope("alpha", {
        from: "2026-09-01T00:00:00Z",
        to: "2026-09-02T00:00:00Z",
      }),
    );
    expect(missing.status).toBe(422);
    expect(portal.requests()).toBe(0);

    const collector = await start(portal.origin, "/portal");
    const redirected = await post(
      collector.socket,
      "/v1/read",
      envelope("alpha", {
        from: "2026-09-01T00:00:00Z",
        to: "2026-09-02T00:00:00Z",
      }),
    );
    expect(redirected.status).toBe(503);
  });
});

function envelope(
  session: string,
  replayRange?: { from: string; to: string },
  cursor?: string,
): unknown {
  return {
    version: 1,
    syncRequest: { ...requestBody, replayRange, cursor },
    storageState: {
      cookies: [
        {
          name: "session",
          value: session,
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
  };
}

async function start(origin: string, entry: string, routes?: unknown[]) {
  const socket = join("/tmp", `wk-${crypto.randomUUID().slice(0, 8)}.sock`);
  const runtime = config(origin, entry, routes);
  runtime.socket = socket;
  const server = await startCollectorServer(runtime);
  cleanup.push(() => server.close());
  return { socket };
}

function config(
  origin: string,
  entry: string,
  routes: unknown[] = [
    { method: "GET", path: entry, action: "entry" },
    { method: "GET", path: "/api/read", action: "read" },
    {
      method: "POST",
      path: "/api/statement",
      action: "request_statement",
    },
  ],
): RuntimeConfig {
  return parseRuntimeConfig({
    version: 1,
    socket: join(tmpdir(), "unused.sock"),
    requestTimeoutMs: 10_000,
    bindings: [
      {
        binding,
        admissionRevision: golden.page.admissionRevision,
        manifest,
        origin,
        routes,
      },
    ],
  });
}

async function startPortal(scenario: string, redirectTarget = "/unknown") {
  let workerRequests = 0;
  let requests = 0;
  let statementRequests = 0;
  let downloadRequests = 0;
  let actionRequests = 0;
  const server = createServer((request, response) => {
    requests++;
    if (request.url === "/portal") {
      if (scenario === "redirect") {
        response.writeHead(302, { location: redirectTarget }).end();
        return;
      }
      if (scenario === "oversized-entry") {
        response.writeHead(200, { "content-type": "text/html" });
        response.write(Buffer.alloc(32 * 1024 * 1024, "x"));
        response.end("x");
        return;
      }
      if (
        scenario === "compressed-oversized-entry" ||
        scenario === "compressed-entry"
      ) {
        const body =
          scenario === "compressed-entry"
            ? Buffer.from("<html><body>ready</body></html>")
            : Buffer.alloc(32 * 1024 * 1024 + 1, "x");
        response
          .writeHead(200, {
            "content-type": "text/html",
            "content-encoding": "gzip",
            "set-cookie": [
              "session=rotated; Path=/; HttpOnly",
              "second=present; Path=/; HttpOnly",
            ],
          })
          .end(gzipSync(body));
        return;
      }
      const behavior =
        scenario === "payment"
          ? "fetch('/api/payment',{method:'POST',body:'{}'}).catch(()=>{})"
          : scenario === "popup"
            ? "window.open('/popup')"
            : scenario === "download"
              ? "const a=document.createElement('a');a.href='/download';a.download='x';a.click()"
              : scenario === "websocket"
                ? "new WebSocket('ws://127.0.0.1:1/socket')"
                : scenario === "serviceworker"
                  ? "navigator.serviceWorker.register('/worker.js').catch(()=>{})"
                  : scenario === "forged-fetch"
                    ? "window.fetch=()=>Promise.resolve({url:location.origin+'/api/read',status:200,body:new Response('{}').body,headers:new Headers()})"
                    : scenario === "statement-hijack"
                      ? "fetch('/api/statement',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({from:'2026-09-01T00:00:00Z',to:'2026-09-02T00:00:00Z'})}).catch(()=>{})"
                      : scenario === "webrtc"
                        ? "if(typeof RTCPeerConnection==='function')fetch('/api/read?webrtc=present').catch(()=>{})"
                        : "";
      response
        .writeHead(200, { "content-type": "text/html" })
        .end(`<html><body><script>${behavior}</script></body></html>`);
      return;
    }
    if (
      request.url === "/api/read" ||
      request.url === "/api/statement" ||
      (scenario === "paged" && request.url === "/api/read?cursor=page-2")
    ) {
      if (request.url === "/api/statement") statementRequests++;
      if (
        scenario === "statement-redirect" &&
        request.url === "/api/statement"
      ) {
        response.writeHead(302, { location: "/api/read" }).end();
        return;
      }
      if (scenario === "mfa" || scenario === "captcha") {
        response
          .writeHead(428, { "x-want-keep-challenge": scenario })
          .end("challenge");
        return;
      }
      if (scenario === "expired") {
        response.writeHead(401).end("expired");
        return;
      }
      if (scenario === "invalid-auth-body") {
        response.writeHead(401).end(Buffer.from([0xff]));
        return;
      }
      if (scenario === "rate-limit") {
        response.writeHead(429, { "retry-after": "60" }).end();
        return;
      }
      const result = structuredClone(golden);
      if (scenario === "paged") {
        const secondPage = request.url === "/api/read?cursor=page-2";
        result.page.cursor = secondPage ? "page-2" : "";
        result.page.complete = secondPage;
        if (secondPage) delete result.page.nextCursor;
        else result.page.nextCursor = "page-2";
      }
      const cookie = request.headers.cookie ?? "";
      if (
        scenario === "compressed-entry" &&
        (!cookie.includes("session=rotated") ||
          !cookie.includes("second=present"))
      ) {
        response.writeHead(401).end();
        return;
      }
      if (scenario === "session")
        (
          result.page.records as Array<{
            balanceSnapshot?: { freshness: string };
          }>
        )[1]!.balanceSnapshot!.freshness = cookie.includes("beta")
          ? "stale"
          : "fresh";
      response
        .writeHead(200, { "content-type": "application/json" })
        .end(JSON.stringify(result));
      return;
    }
    if (request.url === "/worker.js") workerRequests++;
    if (request.url === "/download") downloadRequests++;
    if (["/api/payment", "/popup", "/socket"].includes(request.url ?? ""))
      actionRequests++;
    response.writeHead(200).end("blocked target");
  });
  await listen(server);
  const address = server.address();
  if (address === null || typeof address === "string")
    throw new Error("listen");
  cleanup.push(() => close(server));
  return {
    origin: `http://127.0.0.1:${address.port}`,
    workerRequests: () => workerRequests,
    statementRequests: () => statementRequests,
    downloadRequests: () => downloadRequests,
    actionRequests: () => actionRequests,
    requests: () => requests,
  };
}

async function listen(server: Server): Promise<void> {
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => resolve());
  });
}

async function close(server: Server): Promise<void> {
  await new Promise<void>((resolve) => server.close(() => resolve()));
}

async function post(
  socketPath: string,
  path: string,
  body: unknown,
): Promise<{ status: number; body: string }> {
  const data = JSON.stringify(body);
  return await new Promise((resolve, reject) => {
    const call = request(
      {
        socketPath,
        path,
        method: "POST",
        headers: {
          "content-type": "application/json",
          "content-length": Buffer.byteLength(data),
        },
      },
      (response) => {
        const chunks: Buffer[] = [];
        response.on("data", (chunk: Buffer) => chunks.push(chunk));
        response.on("end", () =>
          resolve({
            status: response.statusCode ?? 0,
            body: Buffer.concat(chunks).toString("utf8"),
          }),
        );
      },
    );
    call.on("error", reject);
    call.end(data);
  });
}
