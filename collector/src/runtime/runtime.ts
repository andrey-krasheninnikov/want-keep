import { createHash } from "node:crypto";

import { chromium, type Browser, type Route } from "playwright";

import {
  deploymentBindingKey,
  parseSyncRequest,
  parseSyncResultJSON,
  type SyncRequest,
  type SyncResult,
} from "../contracts/ingestion.js";
import { ContractError, fail, strictObject } from "../contracts/validation.js";
import type { RouteRule, RuntimeBinding, RuntimeConfig } from "./config.js";

const maximumSessionBytes = 1024 * 1024;
const maximumProviderResponseBytes = 32 * 1024 * 1024;

export interface ReadEnvelope {
  version: 1;
  syncRequest: SyncRequest;
  storageState: unknown;
}

export interface CapabilityEnvelope {
  version: 1;
  binding: SyncRequest["binding"];
  admissionRevision: number;
}

export class CollectorRuntime {
  private browser: Browser | undefined;
  private busy = false;
  private readonly config: RuntimeConfig;

  constructor(config: RuntimeConfig) {
    this.config = config;
  }

  capabilities(value: unknown): unknown {
    const request = parseCapabilityEnvelope(value);
    return structuredClone(this.binding(request).manifest);
  }

  async read(value: unknown, signal: AbortSignal): Promise<SyncResult> {
    if (this.busy) throw new CollectorBusyError();
    signal = AbortSignal.any([
      signal,
      AbortSignal.timeout(this.config.requestTimeoutMs),
    ]);
    this.busy = true;
    try {
      let envelope: ReadEnvelope;
      let binding: RuntimeBinding;
      let entry: RouteRule;
      let target: RouteRule;
      try {
        envelope = parseReadEnvelope(value);
        binding = this.binding({
          version: 1,
          binding: envelope.syncRequest.binding,
          admissionRevision: envelope.syncRequest.admissionRevision,
        });
        try {
          validateStorageState(envelope.storageState, binding.origin);
        } catch (error) {
          if (error instanceof ContractError)
            throw new CollectorSessionInvalidError();
          throw error;
        }
        entry = routeFor(binding, "entry");
        target = routeFor(
          binding,
          envelope.syncRequest.replayRange === undefined ||
            envelope.syncRequest.cursor !== undefined
            ? "read"
            : "request_statement",
        );
      } catch (error) {
        if (error instanceof CollectorPreflightError) throw error;
        if (error instanceof ContractError) throw new CollectorPreflightError();
        throw error;
      }
      if (signal.aborted) throw new Error("request_aborted");
      const browser = await this.getBrowser(signal).catch(() => {
        throw new CollectorBeforeIOUnavailableError();
      });
      try {
        return await readWithContext(
          browser,
          binding,
          entry,
          target,
          envelope.syncRequest,
          envelope.storageState,
          signal,
          this.config.requestTimeoutMs,
        );
      } finally {
        if (signal.aborted && this.browser === browser) {
          this.browser = undefined;
          void browser.close().catch(() => undefined);
        }
      }
    } finally {
      this.busy = false;
    }
  }

  async close(): Promise<void> {
    await this.browser?.close();
    this.browser = undefined;
  }

  private binding(request: CapabilityEnvelope): RuntimeBinding {
    const key = deploymentBindingKey(request.binding);
    const binding = this.config.bindings.find(
      (candidate) =>
        deploymentBindingKey(candidate.binding) === key &&
        candidate.admissionRevision === request.admissionRevision,
    );
    if (binding === undefined) fail();
    return binding;
  }

  private async getBrowser(signal: AbortSignal): Promise<Browser> {
    if (this.browser === undefined || !this.browser.isConnected()) {
      // The pinned runtime accepts signal; LaunchOptions does not declare it.
      const options = {
        headless: true,
        signal,
        timeout: this.config.requestTimeoutMs,
      };
      this.browser = await chromium.launch(options);
    }
    return this.browser;
  }
}

function validateStorageState(value: unknown, origin: string): void {
  const state = strictObject(value, ["cookies", "origins"]);
  if (!Array.isArray(state.cookies) || state.cookies.length > 256) fail();
  const host = new URL(origin).hostname;
  for (const value of state.cookies) {
    const cookie = strictObject(value, [
      "name",
      "value",
      "domain",
      "path",
      "expires",
      "httpOnly",
      "secure",
      "sameSite",
    ]);
    if (
      typeof cookie.name !== "string" ||
      cookie.name.length < 1 ||
      cookie.name.length > 256 ||
      typeof cookie.value !== "string" ||
      cookie.value.length > 8192 ||
      typeof cookie.domain !== "string" ||
      !cookieDomainMatches(cookie.domain, host) ||
      typeof cookie.path !== "string" ||
      !cookie.path.startsWith("/") ||
      typeof cookie.expires !== "number" ||
      !Number.isFinite(cookie.expires) ||
      typeof cookie.httpOnly !== "boolean" ||
      typeof cookie.secure !== "boolean" ||
      !["Lax", "None", "Strict"].includes(cookie.sameSite as string)
    )
      fail();
  }
  if (!Array.isArray(state.origins) || state.origins.length > 8) fail();
  for (const value of state.origins) {
    const stored = strictObject(value, ["origin", "localStorage"]);
    if (stored.origin !== origin || !Array.isArray(stored.localStorage)) fail();
    if (stored.localStorage.length > 256) fail();
    for (const value of stored.localStorage) {
      const item = strictObject(value, ["name", "value"]);
      if (
        typeof item.name !== "string" ||
        item.name.length < 1 ||
        item.name.length > 256 ||
        typeof item.value !== "string" ||
        item.value.length > 8192
      )
        fail();
    }
  }
}

function cookieDomainMatches(domain: string, host: string): boolean {
  const normalized = domain.startsWith(".") ? domain.slice(1) : domain;
  return normalized === host;
}

export class CollectorBusyError extends Error {}
export class CollectorBeforeIOUnavailableError extends Error {}
export class CollectorPreflightError extends ContractError {}
export class CollectorSessionInvalidError extends CollectorPreflightError {}

export function parseCapabilityEnvelope(value: unknown): CapabilityEnvelope {
  const object = strictObject(value, [
    "version",
    "binding",
    "admissionRevision",
  ]);
  if (object.version !== 1) fail();
  const request = parseSyncRequest({
    jobId: "00000000-0000-4000-8000-000000000000",
    attempt: 1,
    leaseToken: "binding-check",
    connectionId: "00000000-0000-4000-8000-000000000001",
    connectionGeneration: 1,
    binding: object.binding,
    admissionRevision: object.admissionRevision,
  });
  return {
    version: 1,
    binding: request.binding,
    admissionRevision: request.admissionRevision,
  };
}

export function parseReadEnvelope(value: unknown): ReadEnvelope {
  const object = strictObject(value, [
    "version",
    "syncRequest",
    "storageState",
  ]);
  if (object.version !== 1 || object.storageState === undefined) fail();
  const encoded = JSON.stringify(object.storageState);
  if (
    encoded === undefined ||
    Buffer.byteLength(encoded, "utf8") < 2 ||
    Buffer.byteLength(encoded, "utf8") > maximumSessionBytes
  )
    fail();
  const state = strictObject(object.storageState, ["cookies", "origins"]);
  if (!Array.isArray(state.cookies) || !Array.isArray(state.origins)) fail();
  return {
    version: 1,
    syncRequest: parseSyncRequest(object.syncRequest),
    storageState: structuredClone(state),
  };
}

async function readWithContext(
  browser: Browser,
  binding: RuntimeBinding,
  entry: RouteRule,
  target: RouteRule,
  request: SyncRequest,
  storageState: unknown,
  signal: AbortSignal,
  timeout: number,
): Promise<SyncResult> {
  if (signal.aborted) throw new CollectorBeforeIOUnavailableError();
  const pendingContext = browser.newContext({
    acceptDownloads: false,
    javaScriptEnabled: false,
    serviceWorkers: "block",
    storageState: storageState as NonNullable<
      Parameters<Browser["newContext"]>[0]
    >["storageState"],
  });
  // Context creation has no native timeout. Dispose late results after cancellation.
  void pendingContext.then(
    (context) => {
      if (signal.aborted) void context.close().catch(() => undefined);
    },
    () => undefined,
  );
  let cancelSetup = (): void => {};
  const cancelled = new Promise<never>((_, reject) => {
    cancelSetup = () => {
      void browser.close().catch(() => undefined);
      reject(new CollectorBeforeIOUnavailableError());
    };
    signal.addEventListener("abort", cancelSetup, { once: true });
    if (signal.aborted) cancelSetup();
  });
  const context = await Promise.race([pendingContext, cancelled])
    .catch(() => {
      throw new CollectorBeforeIOUnavailableError();
    })
    .finally(() => signal.removeEventListener("abort", cancelSetup));
  let violation: Error | undefined;
  const reject = (message: string): void => {
    violation ??= new Error(message);
  };
  const abort = (): void => void context.close();
  signal.addEventListener("abort", abort, { once: true });
  let providerIOStarted = false;
  try {
    if (signal.aborted) throw new Error("request_aborted");
    await context.route("**/*", async (route) => {
      try {
        requireAllowedRequest(binding, request, route);
        const headers = await route.request().allHeaders();
        for (const name of ["host", "connection", "content-length"])
          delete headers[name];
        const response = await fetch(route.request().url(), {
          headers,
          redirect: "error",
          signal,
        });
        const data = await readProviderBody(response);
        const responseHeaders = Object.fromEntries(response.headers);
        // fetch returns decoded bytes; Chromium must not decompress them again.
        delete responseHeaders["content-encoding"];
        delete responseHeaders["content-length"];
        const cookies = response.headers.getSetCookie();
        if (cookies.length > 0)
          responseHeaders["set-cookie"] = cookies.join("\n");
        await route.fulfill({
          status: response.status,
          headers: responseHeaders,
          body: data,
        });
      } catch {
        reject("network_policy_violation");
        await route.abort("blockedbyclient");
      }
    });
    await context.routeWebSocket("**/*", (webSocket) => {
      reject("websocket_blocked");
      webSocket.close();
    });
    const page = await context.newPage();
    context.on("page", (opened) => {
      if (opened !== page) {
        reject("popup_blocked");
        void opened.close();
      }
    });
    page.on("download", (download) => {
      reject("download_blocked");
      void download.cancel().catch(() => undefined);
    });
    try {
      providerIOStarted = true;
      await page.goto(binding.origin + entry.path, {
        waitUntil: "domcontentloaded",
        timeout,
      });
      if (page.url() !== binding.origin + entry.path || violation !== undefined)
        throw violation ?? new Error("redirect_blocked");
      const targetURL = providerTargetURL(binding, target, request);
      const body =
        target.method === "POST"
          ? JSON.stringify({
              from: request.replayRange?.from,
              to: request.replayRange?.to,
            })
          : undefined;
      const cookies = await context.cookies(targetURL);
      const response = await fetch(targetURL, {
        method: target.method,
        body,
        headers: {
          cookie: cookies
            .map((cookie) => `${cookie.name}=${cookie.value}`)
            .join("; "),
          ...(body === undefined ? {} : { "content-type": "application/json" }),
        },
        redirect: "error",
        signal: AbortSignal.any([signal, AbortSignal.timeout(timeout)]),
      });
      const data = await readProviderBody(response);
      if (violation !== undefined) throw violation;
      if (response.url !== targetURL) throw new Error("redirect_blocked");
      if (page.url() !== binding.origin + entry.path)
        throw new Error("navigation_blocked");
      if (
        response.status === 401 ||
        response.status === 403 ||
        response.headers.get("x-want-keep-challenge") === "mfa" ||
        response.headers.get("x-want-keep-challenge") === "captcha"
      ) {
        return providerFailure(request, {
          status: response.status,
          challenge: response.headers.get("x-want-keep-challenge"),
        });
      }
      if (response.status === 429) {
        return providerFailure(request, {
          status: response.status,
          challenge: null,
          retryAfterSeconds: retryAfterSeconds(
            response.headers.get("retry-after"),
          ),
        });
      }
      if (response.status !== 200) throw new Error("provider_read_failed");
      return parseSyncResultJSON(
        new TextDecoder("utf-8", { fatal: true }).decode(data),
        request,
      );
    } finally {
      signal.removeEventListener("abort", abort);
    }
  } catch (error) {
    if (!providerIOStarted) throw new CollectorBeforeIOUnavailableError();
    throw error;
  } finally {
    signal.removeEventListener("abort", abort);
    await context.close().catch(() => undefined);
  }
}

async function readProviderBody(response: Response): Promise<Buffer> {
  if (response.body === null) throw new Error("provider_body_missing");
  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let size = 0;
  while (true) {
    const result = await reader.read();
    if (result.done) break;
    size += result.value.byteLength;
    if (size > maximumProviderResponseBytes) {
      await reader.cancel();
      throw new Error("provider_body_too_large");
    }
    chunks.push(result.value);
  }
  return Buffer.concat(chunks, size);
}

function requireAllowedRequest(
  binding: RuntimeBinding,
  request: SyncRequest,
  route: Route,
): void {
  const providerRequest = route.request();
  const url = new URL(providerRequest.url());
  if (
    url.origin !== binding.origin ||
    url.username !== "" ||
    url.password !== "" ||
    url.hash !== ""
  )
    fail();
  const rule = binding.routes.find(
    (candidate) =>
      candidate.method === providerRequest.method() &&
      candidate.path === url.pathname,
  );
  if (rule === undefined) fail();
  if (rule.action !== "entry") fail();
  if (providerRequest.url() !== providerTargetURL(binding, rule, request))
    fail();
  if (providerRequest.postData() !== null) fail();
}

function providerTargetURL(
  binding: RuntimeBinding,
  rule: RouteRule,
  request: SyncRequest,
): string {
  const url = new URL(rule.path, binding.origin);
  if (rule.action === "read" && request.cursor !== undefined)
    url.searchParams.set("cursor", request.cursor);
  return url.href;
}

function routeFor(
  binding: RuntimeBinding,
  action: RouteRule["action"],
): RouteRule {
  const route = binding.routes.find((candidate) => candidate.action === action);
  if (route === undefined) fail();
  return route;
}

function providerFailure(
  request: SyncRequest,
  response: {
    status: number;
    challenge: string | null;
    retryAfterSeconds?: number;
  },
): SyncResult {
  const kind =
    response.status === 429
      ? "rate_limited"
      : response.challenge === "mfa"
        ? "mfa_required"
        : response.challenge === "captcha"
          ? "captcha_required"
          : "reauthentication_required";
  const data = Buffer.from(
    JSON.stringify({ code: kind, status: response.status }),
    "utf8",
  );
  return {
    outcome: "failure",
    failure: {
      jobId: request.jobId,
      attempt: request.attempt,
      leaseToken: request.leaseToken,
      connectionGeneration: request.connectionGeneration,
      binding: request.binding,
      admissionRevision: request.admissionRevision,
      cursor: request.cursor ?? "",
      kind,
      retryable: kind === "rate_limited",
      ...(response.retryAfterSeconds === undefined
        ? {}
        : { retryAfterSeconds: response.retryAfterSeconds }),
      safeMessage:
        kind === "rate_limited"
          ? "Provider rate limit reached"
          : "Provider authorization is required",
      evidence: [
        {
          id: "provider-state",
          mediaType: "application/json",
          data: data.toString("base64"),
          sha256: createHash("sha256").update(data).digest("hex"),
          locator: "collector:provider-auth-state",
        },
      ],
    },
  };
}

function retryAfterSeconds(value: string | null): number {
  const trimmed = value?.trim();
  const seconds =
    trimmed !== undefined && /^\d+$/.test(trimmed)
      ? Number(trimmed)
      : trimmed === undefined
        ? Number.NaN
        : Math.ceil((Date.parse(trimmed) - Date.now()) / 1000);
  if (!Number.isFinite(seconds)) return 60;
  return Math.min(86_400, Math.max(1, seconds));
}
