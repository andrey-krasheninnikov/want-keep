export { capabilitySupports, parseCapabilityManifest } from "./capabilities.js";
export { SyntheticGateway } from "./synthetic-gateway.js";
export type { CapabilityManifest, SyncRequest, SyncResult } from "./types.js";
export { ContractError } from "./validation.js";
export {
  deploymentBindingKey,
  parseDeploymentBinding,
  parseSyncRequest,
  parseSyncResult,
  parseSyncResultJSON,
} from "./wire.js";
