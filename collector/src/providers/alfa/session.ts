import { fail, strictObject } from "../../contracts/validation.js";

export const sessionHeaderNames = [
  "x-xsrf-token",
  "x-security-request",
  "device-app-id",
  "ao-origin-application-id",
  "zone-offset",
] as const;

export function parseSessionHeaders(value: unknown): Record<string, string> {
  const object = strictObject(value, [...sessionHeaderNames]);
  const result: Record<string, string> = {};
  for (const name of sessionHeaderNames) {
    const value = object[name];
    if (typeof value !== "string" || !/^[\x20-\x7e]{1,2048}$/.test(value))
      fail();
    result[name] = value;
  }
  return result;
}
