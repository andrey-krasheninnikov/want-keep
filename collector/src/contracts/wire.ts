import { createHash } from "node:crypto";

import type { components } from "./generated/ingestion.gen.js";
import {
  recordKinds,
  providers,
  products,
  type SyncRequest,
  type SyncResult,
} from "./types.js";
import {
  DIGEST,
  MAX_ENCODED_RESULT_BYTES,
  MAX_EVIDENCE,
  MAX_EVIDENCE_BYTES,
  MAX_RECORDS,
  RAW_DIGEST,
  array,
  asset,
  boolean,
  date,
  decimal,
  equal,
  evidenceReference,
  fail,
  instant,
  integer,
  nonblankText,
  oneOf,
  optionalText,
  optionalTextOrEmpty,
  parseInstant,
  requiredKeys,
  safeRevision,
  strictBase64,
  strictObject,
  string,
  text,
  textOrEmpty,
  uniqueArray,
  uuid,
} from "./validation.js";

export function parseSyncRequest(value: unknown): SyncRequest {
  const object = strictObject(value, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionId",
    "connectionGeneration",
    "binding",
    "admissionRevision",
    "cursor",
    "replayRange",
  ]);
  requiredKeys(object, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionId",
    "connectionGeneration",
    "binding",
    "admissionRevision",
  ]);
  uuid(object.jobId);
  integer(object.attempt, 1, 100);
  text(object.leaseToken);
  uuid(object.connectionId);
  safeRevision(object.connectionGeneration);
  parseDeploymentBinding(object.binding);
  safeRevision(object.admissionRevision);
  optionalText(object.cursor);
  if (object.replayRange !== undefined) replayRange(object.replayRange);
  return object as SyncRequest;
}

export function parseSyncResultJSON(
  textValue: string,
  expected: SyncRequest,
): SyncResult {
  if (Buffer.byteLength(textValue, "utf8") > MAX_ENCODED_RESULT_BYTES) fail();
  requireIntegerNumberLexemes(textValue);
  let value: unknown;
  try {
    value = JSON.parse(textValue) as unknown;
  } catch {
    fail();
  }
  return parseSyncResult(value, expected);
}

function requireIntegerNumberLexemes(value: string): void {
  let inString = false;
  let escaped = false;
  for (let index = 0; index < value.length; index += 1) {
    const character = value[index]!;
    if (inString) {
      if (escaped) escaped = false;
      else if (character === "\\") escaped = true;
      else if (character === '"') inString = false;
      continue;
    }
    if (character === '"') {
      inString = true;
      continue;
    }
    if (character !== "-" && (character < "0" || character > "9")) continue;
    let end = index + 1;
    while (end < value.length && value[end]! >= "0" && value[end]! <= "9")
      end += 1;
    if (value[end] === "." || value[end] === "e" || value[end] === "E") fail();
    index = end - 1;
  }
}

export function parseSyncResult(
  value: unknown,
  expected: SyncRequest,
): SyncResult {
  let encoded: string | undefined;
  try {
    encoded = JSON.stringify(value);
  } catch {
    fail();
  }
  if (
    encoded === undefined ||
    Buffer.byteLength(encoded, "utf8") > MAX_ENCODED_RESULT_BYTES
  )
    fail();
  const validatedExpected = parseSyncRequest(expected);
  const object = strictObject(value, ["outcome", "page", "failure"]);
  requiredKeys(object, ["outcome"]);
  if (object.outcome === "page") {
    if (object.page === undefined || object.failure !== undefined) fail();
    page(object.page, validatedExpected);
  } else if (object.outcome === "failure") {
    if (object.failure === undefined || object.page !== undefined) fail();
    failure(object.failure, validatedExpected);
  } else {
    fail();
  }
  return object as SyncResult;
}

function page(value: unknown, expected: SyncRequest): void {
  const object = strictObject(value, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionGeneration",
    "binding",
    "admissionRevision",
    "cursor",
    "nextCursor",
    "complete",
    "coverage",
    "evidence",
    "records",
  ]);
  requiredKeys(object, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionGeneration",
    "binding",
    "admissionRevision",
    "cursor",
    "complete",
    "coverage",
    "evidence",
    "records",
  ]);
  echoedToken(object, expected);
  textOrEmpty(object.cursor);
  equal(object.cursor, expected.cursor ?? "");
  optionalText(object.nextCursor);
  boolean(object.complete);
  if (object.complete === true && object.nextCursor !== undefined) fail();
  if (
    object.complete === false &&
    (typeof object.nextCursor !== "string" ||
      object.nextCursor === object.cursor)
  )
    fail();
  coverage(object.coverage);
  const evidenceIDs = evidence(object.evidence);
  let postingCount = 0;
  const descriptors = new Set<string>();
  const balances = new Set<string>();
  const references = new Set<string>();
  array(object.records, 0, MAX_RECORDS).forEach((record) => {
    const summary = ingestionRecord(record, evidenceIDs);
    postingCount += summary.postingCount;
    if (postingCount > MAX_RECORDS) fail();
    if (summary.descriptor !== undefined) {
      if (descriptors.has(summary.descriptor)) fail();
      descriptors.add(summary.descriptor);
    }
    if (summary.balance !== undefined) {
      if (balances.has(summary.balance)) fail();
      balances.add(summary.balance);
    }
    summary.references.forEach((reference) => references.add(reference));
  });
  references.forEach((reference) => {
    if (!descriptors.has(reference)) fail();
  });
}

function failure(value: unknown, expected: SyncRequest): void {
  const object = strictObject(value, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionGeneration",
    "binding",
    "admissionRevision",
    "cursor",
    "kind",
    "retryable",
    "retryAfterSeconds",
    "safeMessage",
    "evidence",
  ]);
  requiredKeys(object, [
    "jobId",
    "attempt",
    "leaseToken",
    "connectionGeneration",
    "binding",
    "admissionRevision",
    "cursor",
    "kind",
    "retryable",
    "evidence",
  ]);
  echoedToken(object, expected);
  textOrEmpty(object.cursor);
  equal(object.cursor, expected.cursor ?? "");
  const kind = string(object.kind);
  oneOf(
    kind,
    new Set([
      "reauthentication_required",
      "mfa_required",
      "captcha_required",
      "rate_limited",
      "temporary_failure",
      "permanent_failure",
      "unsupported_capability",
      "contract_violation",
    ]),
  );
  boolean(object.retryable);
  optionalTextOrEmpty(object.safeMessage);
  if (object.retryAfterSeconds !== undefined)
    integer(object.retryAfterSeconds, 1, 86_400);
  if (
    kind === "rate_limited" &&
    (object.retryable !== true || object.retryAfterSeconds === undefined)
  )
    fail();
  if (kind === "temporary_failure" && object.retryable !== true) fail();
  if (
    [
      "reauthentication_required",
      "mfa_required",
      "captcha_required",
      "permanent_failure",
      "unsupported_capability",
      "contract_violation",
    ].includes(kind) &&
    (object.retryable !== false || object.retryAfterSeconds !== undefined)
  )
    fail();
  evidence(object.evidence);
}

function echoedToken(
  object: Record<string, unknown>,
  expected: SyncRequest,
): void {
  uuid(object.jobId);
  integer(object.attempt, 1, 100);
  text(object.leaseToken);
  safeRevision(object.connectionGeneration);
  parseDeploymentBinding(object.binding);
  safeRevision(object.admissionRevision);
  if (
    object.jobId !== expected.jobId ||
    object.attempt !== expected.attempt ||
    object.leaseToken !== expected.leaseToken ||
    object.connectionGeneration !== expected.connectionGeneration ||
    object.admissionRevision !== expected.admissionRevision ||
    deploymentBindingKey(
      object.binding as components["schemas"]["DeploymentBinding"],
    ) !== deploymentBindingKey(expected.binding)
  )
    fail();
}

interface RecordSummary {
  postingCount: number;
  descriptor?: string;
  balance?: string;
  references: string[];
}

function ingestionRecord(
  value: unknown,
  evidenceIDs: Set<string>,
): RecordSummary {
  const object = strictObject(value, [
    "recordType",
    "account",
    "balanceSnapshot",
    "transaction",
  ]);
  requiredKeys(object, ["recordType"]);
  oneOf(object.recordType, recordKinds);
  const present = [
    object.account,
    object.balanceSnapshot,
    object.transaction,
  ].filter((item) => item !== undefined);
  if (present.length !== 1) fail();
  if (object.recordType === "account" && object.account !== undefined) {
    accountRecord(object.account, evidenceIDs);
    return {
      postingCount: 0,
      descriptor: accountKey(object.account as Record<string, unknown>),
      references: [],
    };
  } else if (
    object.recordType === "balance_snapshot" &&
    object.balanceSnapshot !== undefined
  ) {
    balanceRecord(object.balanceSnapshot, evidenceIDs);
    const reference = accountKey(
      object.balanceSnapshot as Record<string, unknown>,
    );
    return {
      postingCount: 0,
      balance: reference,
      references: [reference],
    };
  } else if (
    object.recordType === "transaction" &&
    object.transaction !== undefined
  ) {
    const postings = transactionRecord(object.transaction, evidenceIDs);
    return {
      postingCount: postings.length,
      references: postings.map((item) => accountKey(item)),
    };
  }
  fail();
}

function accountRecord(value: unknown, evidenceIDs: Set<string>): void {
  const object = strictObject(value, [
    "externalAccountId",
    "product",
    "logNamespace",
    "assetCode",
    "network",
    "name",
    "openingDate",
    "aliases",
    "evidenceId",
  ]);
  requiredKeys(object, [
    "externalAccountId",
    "product",
    "logNamespace",
    "assetCode",
    "name",
    "openingDate",
    "evidenceId",
  ]);
  accountReference(object);
  nonblankText(object.logNamespace);
  nonblankText(object.name);
  date(object.openingDate);
  evidenceReference(object.evidenceId, evidenceIDs);
  if (object.aliases !== undefined)
    array(object.aliases, 0, 32).forEach(cardAlias);
}

function balanceRecord(value: unknown, evidenceIDs: Set<string>): void {
  const object = strictObject(value, [
    "externalAccountId",
    "product",
    "logNamespace",
    "assetCode",
    "network",
    "sourceAsOf",
    "owned",
    "available",
    "locked",
    "debt",
    "creditLimit",
    "ownAvailable",
    "coverage",
    "freshness",
    "evidenceId",
  ]);
  requiredKeys(object, [
    "externalAccountId",
    "product",
    "logNamespace",
    "assetCode",
    "sourceAsOf",
    "owned",
    "available",
    "locked",
    "debt",
    "creditLimit",
    "ownAvailable",
    "coverage",
    "freshness",
    "evidenceId",
  ]);
  accountReference(object);
  nonblankText(object.logNamespace);
  instant(object.sourceAsOf);
  for (const field of ["owned", "available", "locked", "debt", "creditLimit"])
    sourceAmount(object[field], object.assetCode);
  boolean(object.ownAvailable);
  coverage(object.coverage);
  oneOf(object.freshness, new Set(["fresh", "stale", "unknown"]));
  evidenceReference(object.evidenceId, evidenceIDs);
}

function transactionRecord(
  value: unknown,
  evidenceIDs: Set<string>,
): Array<Record<string, unknown>> {
  const object = strictObject(value, [
    "externalAccountId",
    "product",
    "logNamespace",
    "providerRecordId",
    "classification",
    "providerState",
    "economicType",
    "occurredAt",
    "postedAt",
    "merchant",
    "note",
    "feeKnowledge",
    "pnlBasis",
    "evidenceId",
    "postings",
  ]);
  requiredKeys(object, [
    "externalAccountId",
    "product",
    "logNamespace",
    "providerRecordId",
    "classification",
    "providerState",
    "economicType",
    "occurredAt",
    "feeKnowledge",
    "evidenceId",
    "postings",
  ]);
  nonblankText(object.externalAccountId);
  oneOf(object.product, products);
  nonblankText(object.logNamespace);
  nonblankText(object.providerRecordId);
  oneOf(object.classification, new Set(["new", "correction", "ambiguous"]));
  oneOf(
    object.providerState,
    new Set(["unknown", "draft", "pending", "posted", "cancelled", "reversed"]),
  );
  oneOf(
    object.economicType,
    new Set([
      "income",
      "expense",
      "transfer",
      "exchange",
      "refund",
      "yield",
      "trade_result",
    ]),
  );
  instant(object.occurredAt);
  if (object.postedAt !== undefined) instant(object.postedAt);
  optionalTextOrEmpty(object.merchant);
  optionalTextOrEmpty(object.note);
  oneOf(object.feeKnowledge, new Set(["known", "unknown"]));
  if (object.pnlBasis !== undefined)
    oneOf(object.pnlBasis, new Set(["gross", "net"]));
  if (object.economicType !== "trade_result" && object.pnlBasis !== undefined)
    fail();
  evidenceReference(object.evidenceId, evidenceIDs);
  const postings = array(object.postings, 0, MAX_RECORDS).map((value) =>
    strictObject(value, [
      "externalAccountId",
      "product",
      "network",
      "assetCode",
      "money",
      "role",
      "funding",
      "treatment",
    ]),
  );
  postings.forEach(posting);
  return postings;
}

function posting(value: unknown): void {
  const object = strictObject(value, [
    "externalAccountId",
    "product",
    "network",
    "assetCode",
    "money",
    "role",
    "funding",
    "treatment",
  ]);
  requiredKeys(object, [
    "externalAccountId",
    "product",
    "assetCode",
    "money",
    "role",
  ]);
  accountReference(object);
  decimal(object.money);
  oneOf(
    object.role,
    new Set(["principal", "fee", "interest", "funding", "pnl", "reward"]),
  );
  if (object.funding !== undefined)
    oneOf(object.funding, new Set(["own", "credit", "unknown"]));
  if (object.treatment !== undefined)
    oneOf(object.treatment, new Set(["movement", "included", "valuation"]));
}

function evidence(value: unknown): Set<string> {
  const ids = new Set<string>();
  let decodedBytes = 0;
  array(value, 1, MAX_EVIDENCE).forEach((item) => {
    const object = strictObject(item, [
      "id",
      "mediaType",
      "data",
      "sha256",
      "locator",
    ]);
    requiredKeys(object, ["id", "mediaType", "data", "sha256", "locator"]);
    text(object.id, 128);
    const id = object.id;
    if (ids.has(id)) fail();
    ids.add(id);
    oneOf(
      object.mediaType,
      new Set(["application/json", "application/xml", "text/csv", "text/html"]),
    );
    const raw = strictBase64(object.data);
    decodedBytes += raw.length;
    const digest = string(object.sha256);
    if (
      !RAW_DIGEST.test(digest) ||
      createHash("sha256").update(raw).digest("hex") !== digest
    )
      fail();
    nonblankText(object.locator);
  });
  if (decodedBytes > MAX_EVIDENCE_BYTES) fail();
  return ids;
}

function sourceAmount(value: unknown, assetCode: unknown): void {
  const object = strictObject(value, [
    "state",
    "assetCode",
    "amount",
    "reason",
  ]);
  requiredKeys(object, ["state", "assetCode"]);
  asset(object.assetCode);
  equal(object.assetCode, assetCode);
  if (object.state === "known") {
    requiredKeys(object, ["amount"]);
    decimal(object.amount);
    if (object.reason !== undefined) fail();
  } else if (object.state === "unknown" || object.state === "unavailable") {
    requiredKeys(object, ["reason"]);
    nonblankText(object.reason);
    if (object.amount !== undefined) fail();
  } else fail();
}

function coverage(value: unknown): void {
  const object = strictObject(value, ["state", "gaps"]);
  requiredKeys(object, ["state", "gaps"]);
  oneOf(object.state, new Set(["complete", "partial", "unavailable"]));
  const gaps = uniqueArray(object.gaps, 0, 100);
  gaps.forEach((gap) => text(gap));
  if ((object.state === "complete") !== (gaps.length === 0)) fail();
}

export function parseDeploymentBinding(
  value: unknown,
): components["schemas"]["DeploymentBinding"] {
  const object = strictObject(value, [
    "provider",
    "environment",
    "adapterBuildDigest",
    "collectorImageDigest",
    "contractVersion",
    "allowlistRevision",
    "nonSecretConfigRevision",
    "operatorPermissionRevision",
  ]);
  requiredKeys(object, [
    "provider",
    "environment",
    "adapterBuildDigest",
    "collectorImageDigest",
    "contractVersion",
    "allowlistRevision",
    "nonSecretConfigRevision",
    "operatorPermissionRevision",
  ]);
  oneOf(object.provider, providers);
  text(object.environment, 128);
  if (
    typeof object.adapterBuildDigest !== "string" ||
    !DIGEST.test(object.adapterBuildDigest)
  )
    fail();
  if (
    typeof object.collectorImageDigest !== "string" ||
    !DIGEST.test(object.collectorImageDigest)
  )
    fail();
  equal(object.contractVersion, "10");
  for (const field of [
    "allowlistRevision",
    "nonSecretConfigRevision",
    "operatorPermissionRevision",
  ] as const)
    text(object[field], 128);
  return object as components["schemas"]["DeploymentBinding"];
}

export function deploymentBindingKey(
  value: components["schemas"]["DeploymentBinding"],
): string {
  return JSON.stringify([
    value.provider,
    value.environment,
    value.adapterBuildDigest,
    value.collectorImageDigest,
    value.contractVersion,
    value.allowlistRevision,
    value.nonSecretConfigRevision,
    value.operatorPermissionRevision,
  ]);
}

function replayRange(value: unknown): void {
  const object = strictObject(value, ["from", "to"]);
  requiredKeys(object, ["from", "to"]);
  const from = parseInstant(object.from);
  const to = parseInstant(object.to);
  if (from >= to || to - from > 90n * 24n * 60n * 60n * 1_000_000_000n) fail();
}

function accountReference(object: Record<string, unknown>): void {
  nonblankText(object.externalAccountId);
  oneOf(object.product, products);
  asset(object.assetCode);
  optionalTextOrEmpty(object.network);
}

function accountKey(object: Record<string, unknown>): string {
  return JSON.stringify([
    object.externalAccountId,
    object.product,
    object.network ?? "",
    object.assetCode,
  ]);
}

function cardAlias(value: unknown): void {
  const object = strictObject(value, ["id", "label", "lastFour"]);
  requiredKeys(object, ["id", "label", "lastFour"]);
  nonblankText(object.id);
  nonblankText(object.label, 100);
  if (
    typeof object.lastFour !== "string" ||
    !/^[0-9]{4}$/.test(object.lastFour)
  )
    fail();
  const digits = Array.from(object.label as string).filter((character) =>
    /\p{Decimal_Number}/u.test(character),
  );
  if (
    digits.some((digit) => !/[0-9]/.test(digit)) ||
    (digits.length !== 0 && digits.join("") !== object.lastFour)
  )
    fail();
}
