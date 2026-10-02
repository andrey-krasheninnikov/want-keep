import { loadRuntimeConfig } from "./runtime/config.js";
import { startCollectorServer } from "./runtime/server.js";

const path = process.env.WANT_KEEP_COLLECTOR_CONFIG_FILE;
if (path === undefined)
  throw new Error("WANT_KEEP_COLLECTOR_CONFIG_FILE is required");
const server = await startCollectorServer(loadRuntimeConfig(path));
for (const signal of ["SIGINT", "SIGTERM"] as const)
  process.once(
    signal,
    () => void server.close().finally(() => process.exit(0)),
  );
