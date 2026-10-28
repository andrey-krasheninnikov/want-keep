import { describe, expect, it } from "vitest";
import { alfaManifest } from "../../src/providers/alfa/read.js";
import { parseSessionHeaders } from "../../src/providers/alfa/session.js";
import { parseRuntimeConfig } from "../../src/runtime/config.js";
import { parseReadEnvelope } from "../../src/runtime/runtime.js";

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
};
const headers = {
  "x-xsrf-token": "synthetic-xsrf",
  "x-security-request": "synthetic-security",
  "device-app-id": "synthetic-device",
  "ao-origin-application-id": "synthetic-app",
  "zone-offset": "180",
};
const config = {
  version: 1,
  socket: "/tmp/synthetic.sock",
  requestTimeoutMs: 1000,
  bindings: [
    {
      binding,
      admissionRevision: 1,
      manifest: alfaManifest,
      origin: "http://127.0.0.1:1234",
      routes: [
        { method: "GET", action: "entry", path: "/history/" },
        { method: "GET", action: "read", path: "/api/v1/account/" },
      ],
    },
  ],
};
const envelope = {
  version: 1,
  syncRequest: {
    jobId: "11111111-1111-4111-8111-111111111111",
    connectionId: "22222222-2222-4222-8222-222222222222",
    attempt: 1,
    leaseToken: "synthetic-lease",
    connectionGeneration: 1,
    binding,
    admissionRevision: 1,
  },
  storageState: { cookies: [], origins: [] },
  sessionHeaders: headers,
};

describe("Alfa closed contract", () => {
  it("accepts only build-owned origins, manifest and routes", () => {
    expect(parseRuntimeConfig(config).bindings[0]?.manifest).toEqual(
      alfaManifest,
    );
    for (const mutate of [
      (c: typeof config) => {
        c.bindings[0]!.binding.allowlistRevision = "other";
      },
      (c: typeof config) => {
        c.bindings[0]!.routes[1]!.path = "/api/payment";
      },
      (c: typeof config) => {
        c.bindings[0]!.manifest.products = ["credit_card"];
      },
    ]) {
      const invalid = structuredClone(config);
      mutate(invalid);
      expect(() => parseRuntimeConfig(invalid)).toThrow();
    }
  });
  it("bounds credential headers and rejects instructions from jobs", () => {
    expect(parseReadEnvelope(envelope).sessionHeaders).toEqual(headers);
    expect(() =>
      parseSessionHeaders({ ...headers, authorization: "synthetic-secret" }),
    ).toThrow();
    expect(() =>
      parseSessionHeaders({ ...headers, "x-xsrf-token": "a\r\nCookie: b" }),
    ).toThrow();
    expect(() =>
      parseReadEnvelope({ ...envelope, url: "https://foreign.invalid" }),
    ).toThrow();
    expect(() =>
      parseReadEnvelope({ ...envelope, sessionHeaders: undefined }),
    ).toThrow();
    expect(() =>
      parseReadEnvelope({
        ...envelope,
        syncRequest: {
          ...envelope.syncRequest,
          binding: { ...binding, provider: "bybit" },
        },
      }),
    ).toThrow();
  });
});
