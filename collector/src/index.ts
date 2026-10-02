export {
  collectorEnvironments,
  parseCollectorEnvironment,
  type CollectorEnvironment,
} from "./config/environment.js";
export {
  capabilitySupports,
  ContractError,
  parseCapabilityManifest,
  parseDeploymentBinding,
  parseSyncRequest,
  parseSyncResult,
  parseSyncResultJSON,
  SyntheticGateway,
  type CapabilityManifest,
  type SyncRequest,
  type SyncResult,
} from "./contracts/ingestion.js";
