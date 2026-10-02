import { chmodSync, lstatSync, rmSync } from "node:fs";
import {
  createServer,
  type IncomingMessage,
  type ServerResponse,
} from "node:http";
import { createConnection, type Server } from "node:net";

import { ContractError } from "../contracts/ingestion.js";
import type { RuntimeConfig } from "./config.js";
import {
  CollectorBusyError,
  CollectorLaunchUnavailableError,
  CollectorPreflightError,
  CollectorSessionInvalidError,
  CollectorRuntime,
} from "./runtime.js";

const maximumRequestBytes = 2 * 1024 * 1024;

export async function startCollectorServer(
  config: RuntimeConfig,
): Promise<{ close(): Promise<void> }> {
  await removeStaleSocket(config.socket);
  const runtime = new CollectorRuntime(config);
  const server = createServer((request, response) => {
    void handle(runtime, request, response, config.requestTimeoutMs);
  });
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(config.socket, () => {
      server.off("error", reject);
      resolve();
    });
  });
  chmodSync(config.socket, 0o600);
  return {
    async close() {
      await closeServer(server);
      await runtime.close();
      rmSync(config.socket, { force: true });
    },
  };
}

async function handle(
  runtime: CollectorRuntime,
  request: IncomingMessage,
  response: ServerResponse,
  timeout: number,
): Promise<void> {
  response.setHeader("cache-control", "no-store");
  response.setHeader("content-type", "application/json");
  response.setHeader("x-content-type-options", "nosniff");
  if (request.method === "GET" && request.url === "/ready") {
    response.setHeader("content-type", "text/plain; charset=utf-8");
    response.end("want-keep-browser-collector/1\n");
    return;
  }
  if (
    request.method !== "POST" ||
    (request.url !== "/v1/capabilities" && request.url !== "/v1/read") ||
    request.headers["content-type"]?.split(";", 1)[0] !== "application/json"
  ) {
    response.writeHead(404).end('{"code":"not_found"}');
    return;
  }
  const controller = new AbortController();
  const timer = setTimeout(() => {
    controller.abort();
    request.destroy(new Error("request_timeout"));
  }, timeout);
  request.once("aborted", () => controller.abort());
  try {
    const value = await readJSON(request);
    const result =
      request.url === "/v1/capabilities"
        ? runtime.capabilities(value)
        : await runtime.read(value, controller.signal);
    if (controller.signal.aborted) throw new Error("request_aborted");
    response.end(JSON.stringify(result));
  } catch (error) {
    const status =
      error instanceof CollectorBusyError
        ? 409
        : error instanceof ContractError
          ? 422
          : 503;
    if (!response.headersSent)
      response.writeHead(status).end(
        JSON.stringify({
          code:
            status === 409
              ? "collector_busy"
              : status === 422
                ? error instanceof CollectorSessionInvalidError
                  ? "collector_session_invalid"
                  : error instanceof CollectorPreflightError
                    ? "collector_preflight_rejected"
                    : "invalid_contract"
                : error instanceof CollectorLaunchUnavailableError
                  ? "collector_launch_unavailable"
                  : "collector_unavailable",
        }),
      );
  } finally {
    clearTimeout(timer);
  }
}

async function readJSON(request: IncomingMessage): Promise<unknown> {
  const chunks: Buffer[] = [];
  let size = 0;
  for await (const raw of request) {
    const chunk = Buffer.from(raw);
    size += chunk.length;
    if (size > maximumRequestBytes) throw new ContractError();
    chunks.push(chunk);
  }
  const text = Buffer.concat(chunks).toString("utf8");
  if (text.length < 2) throw new ContractError();
  try {
    return JSON.parse(text) as unknown;
  } catch {
    throw new ContractError();
  }
}

async function removeStaleSocket(path: string): Promise<void> {
  try {
    if (!lstatSync(path).isSocket()) throw new Error("socket path is occupied");
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code === "ENOENT") return;
    if ((error as Error).message === "socket path is occupied") throw error;
  }
  const active = await new Promise<boolean>((resolve) => {
    const socket = createConnection(path);
    socket.once("connect", () => {
      socket.destroy();
      resolve(true);
    });
    socket.once("error", () => resolve(false));
  });
  if (active) throw new Error("collector is already running");
  rmSync(path);
}

async function closeServer(server: Server): Promise<void> {
  await new Promise<void>((resolve, reject) => {
    server.close((error) => (error === undefined ? resolve() : reject(error)));
  });
}
