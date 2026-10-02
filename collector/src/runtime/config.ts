import { readFileSync } from "node:fs";

import type { components } from "../contracts/generated/ingestion.gen.js";
import {
  deploymentBindingKey,
  parseCapabilityManifest,
  parseDeploymentBinding,
  type CapabilityManifest,
} from "../contracts/ingestion.js";
import {
  array,
  fail,
  integer,
  requiredKeys,
  strictObject,
  text,
} from "../contracts/validation.js";

export type DeploymentBinding = components["schemas"]["DeploymentBinding"];
export type RouteAction = "entry" | "read" | "request_statement";

export interface RouteRule {
  method: "GET" | "POST";
  path: string;
  action: RouteAction;
}

export interface RuntimeBinding {
  binding: DeploymentBinding;
  admissionRevision: number;
  manifest: CapabilityManifest;
  origin: string;
  routes: RouteRule[];
}

export interface RuntimeConfig {
  version: 1;
  socket: string;
  requestTimeoutMs: number;
  bindings: RuntimeBinding[];
}

const allowedActions = new Set<RouteAction>([
  "entry",
  "read",
  "request_statement",
]);

export function loadRuntimeConfig(path: string): RuntimeConfig {
  if (path.length < 1 || path.length > 4096) fail();
  const raw = readFileSync(path, "utf8");
  if (Buffer.byteLength(raw, "utf8") > 1024 * 1024) fail();
  return parseRuntimeConfig(JSON.parse(raw) as unknown);
}

export function parseRuntimeConfig(value: unknown): RuntimeConfig {
  const root = strictObject(value, [
    "version",
    "socket",
    "requestTimeoutMs",
    "bindings",
  ]);
  requiredKeys(root, ["version", "socket", "requestTimeoutMs", "bindings"]);
  if (root.version !== 1) fail();
  text(root.socket, 4096);
  if (!(root.socket as string).startsWith("/")) fail();
  integer(root.requestTimeoutMs, 1_000, 120_000);
  const seen = new Set<string>();
  const bindings = array(root.bindings, 1, 32).map(parseRuntimeBinding);
  for (const binding of bindings) {
    const key = deploymentBindingKey(binding.binding);
    if (seen.has(key)) fail();
    seen.add(key);
  }
  return {
    version: 1,
    socket: root.socket as string,
    requestTimeoutMs: root.requestTimeoutMs as number,
    bindings,
  };
}

function parseRuntimeBinding(value: unknown): RuntimeBinding {
  const object = strictObject(value, [
    "binding",
    "admissionRevision",
    "manifest",
    "origin",
    "routes",
  ]);
  requiredKeys(object, [
    "binding",
    "admissionRevision",
    "manifest",
    "origin",
    "routes",
  ]);
  const binding = parseDeploymentBinding(object.binding);
  integer(object.admissionRevision, 1, Number.MAX_SAFE_INTEGER);
  const manifest = parseCapabilityManifest(object.manifest);
  if (manifest.provider !== binding.provider) fail();
  const origin = parseOrigin(object.origin, binding.environment);
  const routes = array(object.routes, 2, 16).map(parseRoute);
  const routeKeys = new Set<string>();
  const actionCounts = new Map<RouteAction, number>();
  for (const route of routes) {
    const key = `${route.method} ${route.path}`;
    if (routeKeys.has(key)) fail();
    routeKeys.add(key);
    actionCounts.set(route.action, (actionCounts.get(route.action) ?? 0) + 1);
  }
  if (actionCounts.get("entry") !== 1 || actionCounts.get("read") !== 1) fail();
  if ((actionCounts.get("request_statement") ?? 0) > 1) fail();
  return {
    binding,
    admissionRevision: object.admissionRevision as number,
    manifest,
    origin,
    routes,
  };
}

function parseOrigin(value: unknown, environment: string): string {
  text(value, 512);
  let url: URL;
  try {
    url = new URL(value as string);
  } catch {
    fail();
  }
  if (
    url.origin !== value ||
    url.username !== "" ||
    url.password !== "" ||
    url.search !== "" ||
    url.hash !== ""
  )
    fail();
  if (environment === "production") {
    if (url.protocol !== "https:") fail();
  } else if (url.protocol !== "http:" || !isLoopback(url.hostname)) {
    fail();
  }
  return url.origin;
}

function isLoopback(hostname: string): boolean {
  return (
    hostname === "localhost" || hostname === "127.0.0.1" || hostname === "[::1]"
  );
}

function parseRoute(value: unknown): RouteRule {
  const object = strictObject(value, ["method", "path", "action"]);
  requiredKeys(object, ["method", "path", "action"]);
  if (object.method !== "GET" && object.method !== "POST") fail();
  text(object.path, 512);
  const path = object.path as string;
  if (!path.startsWith("/") || path.includes("?") || path.includes("#")) fail();
  const resolved = new URL(path, "https://collector.invalid");
  if (
    resolved.origin !== "https://collector.invalid" ||
    resolved.pathname !== path
  )
    fail();
  if (
    typeof object.action !== "string" ||
    !allowedActions.has(object.action as RouteAction)
  )
    fail();
  const action = object.action as RouteAction;
  if ((action === "entry" || action === "read") && object.method !== "GET")
    fail();
  if (action === "request_statement" && object.method !== "POST") fail();
  return { method: object.method, path, action };
}
