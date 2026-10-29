import { readFile } from "node:fs/promises";
import { expect, test } from "@playwright/test";
import { authenticator } from "./access/authenticator";
import { AccountWriteFault } from "./access/database-fault";

test("closed household, separate passkeys, cash command, invitation and personal recovery", async ({
  browser,
}) => {
  test.setTimeout(150_000);
  const origin = process.env.WANT_KEEP_ACCESS_ORIGIN!;
  const bootstrapPath = process.env.WANT_KEEP_ACCESS_BOOTSTRAP_FILE;
  if (!bootstrapPath || process.env.WANT_KEEP_ACCESS_E2E !== "1")
    throw new Error(
      "Run make e2e SCENARIO=access with isolated PostgreSQL and API",
    );
  const first = await browser.newContext({
    baseURL: origin,
    locale: "en-US",
    viewport: { width: 1440, height: 900 },
  });
  const second = await browser.newContext({
    baseURL: origin,
    locale: "en-US",
    viewport: { width: 1280, height: 720 },
  });
  const page = await first.newPage();
  const partner = await second.newPage();
  await page.clock.install({ time: new Date() });
  let keyA = await authenticator(first, page);
  await authenticator(second, partner);
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  partner.on("pageerror", (error) => errors.push(error.message));
  try {
    await page.goto("/setup");
    await page
      .getByLabel("Operator token")
      .fill((await readFile(bootstrapPath, "utf8")).trim());
    await page.getByLabel("Your name", { exact: true }).fill("Alex Test");
    await page
      .getByLabel("Household name", { exact: true })
      .fill("Synthetic household");
    await page
      .getByLabel("Household timezone", { exact: true })
      .fill("Europe/Moscow");
    let bootstrapFinishes = 0;
    await page.route("**/api/v1/auth/enrollment/verify", async (route) => {
      bootstrapFinishes++;
      await route.fetch();
      await route.abort("failed");
    });
    await page
      .getByRole("button", {
        name: "Create household and passkey",
        exact: true,
      })
      .click();
    await expect(
      page.getByText("No response received.", { exact: false }),
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Check again", exact: true })
      .click();
    await expect(
      page.getByRole("link", { name: "Set up tracking", exact: true }),
    ).toBeVisible();
    expect(bootstrapFinishes).toBe(1);
    await page.unroute("**/api/v1/auth/enrollment/verify");
    await page.goto("/settings/security");
    await page
      .getByRole("button", { name: "Issue new codes", exact: true })
      .click();
    await expect(
      page.getByRole("heading", { name: "Save your personal recovery codes" }),
    ).toBeVisible();
    const recoveryCode = await page
      .locator(".recovery-codes code")
      .first()
      .innerText();
    await page.getByRole("button", { name: "Codes saved — continue" }).click();
    await page.goto("/onboarding");
    await expect(page).toHaveURL(/\/onboarding$/);
    await expect(
      page.getByText("Alex Test", { exact: true }).last(),
    ).toBeVisible();

    await page
      .getByLabel("Account name", { exact: true })
      .fill("Cash at start");
    await page
      .getByRole("textbox", { name: "Tracking start date", exact: true })
      .fill("09/01/2026");
    await page
      .getByLabel("Opening balance", { exact: true })
      .fill("12345678901234567890.123456789012345678");
    await page.getByRole("button", { name: "RU", exact: true }).click();
    await expect(
      page.getByLabel("Название счёта", { exact: true }),
    ).toHaveValue("Cash at start");
    await expect(
      page.getByRole("textbox", { name: "Дата начала учёта", exact: true }),
    ).toHaveValue("01.09.2026");
    await page.getByRole("button", { name: "EN", exact: true }).click();
    let accountPosts = 0;
    const commandKeys: (string | undefined)[] = [];
    const writeFault = new AccountWriteFault();
    writeFault.install();
    await page.route("**/api/v1/accounts", async (route) => {
      if (route.request().method() !== "POST") {
        await route.continue();
        return;
      }
      accountPosts++;
      commandKeys.push(route.request().headers()["idempotency-key"]);
      const response = await route.fetch();
      if (accountPosts <= 2) {
        expect(response.status()).toBe(500);
        if (accountPosts === 2) writeFault.remove();
      } else if (accountPosts === 3) {
        expect(response.status()).toBe(409);
        await route.fulfill({ response });
        return;
      } else expect(response.ok()).toBe(true);
      await route.abort("failed");
    });
    await page
      .getByRole("button", { name: "Add account", exact: true })
      .click();
    await expect(
      page.getByRole("button", {
        name: "Retry with the original key",
        exact: true,
      }),
    ).toBeVisible();
    expect(accountPosts).toBe(1);
    await page
      .getByRole("button", { name: "Retry with the original key", exact: true })
      .click();
    await expect(
      page.getByRole("button", {
        name: "Retry with the original key",
        exact: true,
      }),
    ).toBeEnabled();
    expect(accountPosts).toBe(2);
    await page.reload();
    await page
      .getByRole("button", {
        name: "Restore the original request",
        exact: true,
      })
      .click();
    await page
      .getByLabel("Account name", { exact: true })
      .fill("Cash at start");
    await page
      .getByRole("textbox", { name: "Tracking start date", exact: true })
      .fill("09/01/2026");
    await page.getByLabel("Opening balance", { exact: true }).fill("999");
    await page
      .getByRole("button", { name: "Retry with the original key", exact: true })
      .click();
    await expect(page.getByRole("alert")).toContainText(
      "The fields differ from the original request",
    );
    await page
      .getByLabel("Opening balance", { exact: true })
      .fill("12345678901234567890.123456789012345678");
    await page
      .getByRole("button", { name: "Retry with the original key", exact: true })
      .click();
    await expect(
      page.getByRole("heading", { name: "Account added: Cash at start" }),
    ).toBeVisible();
    expect(accountPosts).toBe(4);
    expect(commandKeys[0]).toBeTruthy();
    expect(new Set(commandKeys).size).toBe(1);
    await expect(page.locator(".access-notice .access-amount")).toHaveText(
      "12,345,678,901,234,567,890.123456789012345678 RUB",
    );
    await page.unroute("**/api/v1/accounts");
    await page.reload();
    await expect(page.getByText("Cash at start", { exact: true })).toHaveCount(
      1,
    );
    await page
      .getByRole("button", { name: "Create invitation", exact: true })
      .click();
    const link = page.getByLabel("Private invitation link", { exact: true });
    await expect(link).toBeVisible();
    const invitation = await link.inputValue();
    await partner.goto(invitation);
    await expect(partner).toHaveURL(`${origin}/invite`);
    await partner
      .getByRole("button", { name: "Check invitation", exact: true })
      .click();
    await expect(
      partner.getByRole("heading", { name: "Synthetic household" }),
    ).toBeVisible();
    await partner.getByLabel("Your name", { exact: true }).fill("Sam Test");
    let invitationFinishes = 0;
    await partner.route("**/api/v1/invitations/accept", async (route) => {
      invitationFinishes++;
      await route.fetch();
      await route.abort("failed");
    });
    await partner
      .getByRole("button", { name: "Join with your own passkey", exact: true })
      .click();
    await expect(
      partner.getByText("No response received.", { exact: false }),
    ).toBeVisible();
    await partner
      .getByRole("button", { name: "Check again", exact: true })
      .click();
    await expect(
      partner.getByRole("link", { name: "Set up tracking", exact: true }),
    ).toBeVisible();
    expect(invitationFinishes).toBe(1);
    await partner.unroute("**/api/v1/invitations/accept");
    await partner.goto("/onboarding");
    await expect(partner).toHaveURL(/\/onboarding$/);
    await expect(
      partner.getByText("Cash at start", { exact: true }),
    ).toBeVisible();

    const me = await page.request.get("/api/v1/me");
    expect(me.status()).toBe(200);
    const unchanged = (await me.json()).session.idleExpiresAt;
    expect(
      (await (await page.request.get("/api/v1/me")).json()).session
        .idleExpiresAt,
    ).toBe(unchanged);
    const denied = await page.request.post("/api/v1/auth/session/activity", {
      data: {},
      headers: { Origin: origin },
    });
    expect(denied.status()).toBe(401);
    const wrongOrigin = await page.request.post("/api/v1/auth/login/options", {
      data: { purpose: "login" },
      headers: { Origin: "https://unrelated.invalid" },
    });
    expect(wrongOrigin.status()).toBe(401);
    const cookies = await first.cookies();
    const session = cookies.find(
      (cookie) => cookie.name === "want_keep_session",
    );
    expect(session).toMatchObject({
      httpOnly: true,
      secure: true,
      sameSite: "Lax",
    });
    await expect(
      page.evaluate(() => Object.keys(localStorage)),
    ).resolves.toEqual(["want-keep.locale"]);

    await page.getByRole("button", { name: "Sign out", exact: true }).click();
    await expect(page).toHaveURL(/\/login$/);
    await page.getByRole("link", { name: "Trouble signing in?" }).click();
    await page.getByLabel("Personal recovery code").fill(recoveryCode);
    await page.getByRole("button", { name: "Check code", exact: true }).click();
    await expect(
      page.getByRole("button", { name: "Create a new passkey" }),
    ).toBeVisible();
    await keyA.cdp.send("WebAuthn.removeVirtualAuthenticator", {
      authenticatorId: keyA.id,
    });
    keyA = await authenticator(first, page);
    await page.getByRole("button", { name: "Create a new passkey" }).click();
    await expect(
      page.getByRole("heading", { name: "Save your personal recovery codes" }),
    ).toBeVisible();
    await page.getByRole("button", { name: "Codes saved — continue" }).click();
    expect((await partner.request.get("/api/v1/me")).status()).toBe(200);
    await page.getByRole("button", { name: "Sign out", exact: true }).click();
    await page.route("**/api/v1/auth/login/options", async (route) => {
      const response = await route.fetch();
      const options = await response.json();
      await route.fulfill({
        response,
        json: { ...options, rpId: "unrelated.invalid" },
      });
    });
    await page
      .getByRole("button", { name: "Log in with Passkeys", exact: true })
      .click();
    await expect(
      page.getByText("Passkey confirmation failed.", { exact: false }),
    ).toBeVisible();
    await page.unroute("**/api/v1/auth/login/options");
    await page
      .getByRole("button", { name: "Log in with Passkeys", exact: true })
      .click();
    await expect(page).toHaveURL(/\/overview$/);
    await page.goto("/settings/security");
    await page
      .getByRole("button", { name: "Issue new codes", exact: true })
      .click();
    await expect(page.locator(".recovery-codes code")).toHaveCount(10);
    await page.getByRole("button", { name: "Codes saved — continue" }).click();

    for (const viewport of [
      { width: 1280, height: 720 },
      { width: 1440, height: 900 },
      { width: 640, height: 360 },
      { width: 720, height: 450 },
    ]) {
      await page.setViewportSize(viewport);
      await page.goto("/onboarding");
      await expect(
        page.getByRole("heading", { name: "Add cash funds" }),
      ).toBeVisible();
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
      ).toBe(true);
      await page
        .getByRole("button", { name: "Add account", exact: true })
        .scrollIntoViewIfNeeded();
      await expect(
        page.getByRole("button", { name: "Add account", exact: true }),
      ).toBeInViewport();
    }
    await page.emulateMedia({ reducedMotion: "reduce" });
    await page.getByLabel("Account name", { exact: true }).fill("Unsaved cash");
    await page.getByRole("link", { name: "Plan", exact: true }).click();
    await expect(page.getByRole("alertdialog")).toBeVisible();
    await page.getByRole("button", { name: "Stay", exact: true }).click();
    await expect(
      page.getByRole("link", { name: "Plan", exact: true }),
    ).toBeFocused();
    await page.getByRole("link", { name: "Plan", exact: true }).click();
    await expect(page.getByRole("alertdialog")).toBeVisible();
    await page.getByRole("alertdialog").press("Escape");
    await expect(
      page.getByRole("link", { name: "Plan", exact: true }),
    ).toBeFocused();
    await expect(page.getByLabel("Account name", { exact: true })).toHaveValue(
      "Unsaved cash",
    );
    await first.setOffline(true);
    await expect(
      page.getByText("No connection.", { exact: false }),
    ).toBeVisible();
    await first.setOffline(false);
    await page.clock.fastForward(31 * 60_000);
    await expect(page).toHaveURL(/\/login$/);
    await expect(
      page.getByRole("textbox", { name: "Account name", exact: true }),
    ).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: "Log in with Passkeys", exact: true }),
    ).toBeVisible();
    // Only the tab clock expired: the real server cookie is still active.
    await page.clock.setSystemTime(new Date());
    let unnecessaryLogin = 0;
    page.on("request", (request) => {
      if (request.url().endsWith("/auth/login/options")) unnecessaryLogin++;
    });
    await page
      .getByRole("button", { name: "Log in with Passkeys", exact: true })
      .click();
    await expect(page).toHaveURL(/\/onboarding$/);
    await expect(page.getByLabel("Account name", { exact: true })).toHaveValue(
      "Unsaved cash",
    );
    expect(unnecessaryLogin).toBe(0);
    expect(errors).toEqual([]);
  } finally {
    await first.close();
    await second.close();
  }
});
