import { expect, test } from "@playwright/test";

import { contrastPairs } from "../src/design-system/contrast-pairs";
import { ThemeContract } from "../scripts/design/theme-contract";

for (const viewport of [
  { width: 1280, height: 720 },
  { width: 1440, height: 900 },
  { width: 640, height: 360 },
  { width: 720, height: 450 },
]) {
  test(`fonts, exact amounts and allowed pairs at ${viewport.width}`, async ({
    page,
  }, testInfo) => {
    await page.setViewportSize(viewport);
    const errors: string[] = [];
    const externalFonts: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("request", (request) => {
      if (
        request.resourceType() === "font" &&
        new URL(request.url()).origin !== "http://127.0.0.1:4180"
      )
        externalFonts.push(request.url());
    });
    await page.goto("/__design/tokens");
    await expect(
      page.getByRole("heading", { name: "Токены и типографика" }),
    ).toBeVisible();
    await page.evaluate(async () => {
      await document.fonts.ready;
    });
    await page.screenshot({
      path: testInfo.outputPath(`design-tokens-${viewport.width}.png`),
      fullPage: true,
    });
    expect(
      await page.evaluate(() =>
        [...document.fonts].map((font) => [
          font.family.replace(/^["']|["']$/g, ""),
          font.status,
        ]),
      ),
    ).toEqual(
      expect.arrayContaining([
        ["Manrope Variable", "loaded"],
        ["Days One", "loaded"],
      ]),
    );
    await expect(page.getByTestId("amount-ETH")).toHaveText(
      "0.123456789012345678",
    );
    await expect(page.getByTestId("amount-USDC")).toHaveText(
      "0.123456789012345678",
    );
    await page.getByText("Предельная длина: 256 символов").click();
    await expect(page.getByTestId("long-amount")).toHaveText(
      "9".repeat(237) + ".123456789012345678",
    );
    expect(
      await page
        .getByTestId("long-amount")
        .evaluate((element) => element.scrollWidth <= element.clientWidth),
    ).toBe(true);
    const tokens = await page.evaluate(
      (names) =>
        Object.fromEntries(
          names.map((name) => [
            name,
            getComputedStyle(document.documentElement)
              .getPropertyValue(name)
              .trim(),
          ]),
        ),
      [
        ...new Set(
          contrastPairs.flatMap((pair) => [pair.foreground, pair.background]),
        ),
      ],
    );
    for (const pair of contrastPairs) {
      if (pair.minimum !== null)
        expect(
          ThemeContract.contrast(
            tokens[pair.foreground],
            tokens[pair.background],
          ),
          pair.id,
        ).toBeGreaterThanOrEqual(pair.minimum);
    }
    const widths = await page
      .getByTestId("tabular-sample")
      .evaluate((element) => {
        return ["1111111111", "8888888888", "0000000000"].map((text) => {
          const span = document.createElement("span");
          span.textContent = text;
          element.append(span);
          const width = span.getBoundingClientRect().width;
          span.remove();
          return width;
        });
      });
    expect(Math.max(...widths) - Math.min(...widths)).toBeLessThan(0.1);
    expect(externalFonts).toEqual([]);
    expect(errors).toEqual([]);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
  });
}

test("keyboard feedback, locale, system light preference and reduced motion", async ({
  page,
}) => {
  await page.emulateMedia({ colorScheme: "light", reducedMotion: "reduce" });
  await page.goto("/__design/tokens");
  await expect(page.locator("html")).toHaveCSS(
    "background-color",
    "rgb(20, 20, 20)",
  );
  await expect(
    page.getByRole("button", { name: "English", exact: true }),
  ).toBeVisible();
  await page.keyboard.press("Tab");
  await expect(
    page.getByRole("button", { name: "English", exact: true }),
  ).toBeFocused();
  await expect(
    page.getByRole("button", { name: "English", exact: true }),
  ).toHaveCSS("background-color", "rgb(37, 37, 37)");
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("heading", { name: "Tokens and typography" }),
  ).toBeVisible();
  const button = page.getByRole("button", {
    name: "Normal / Hover / Active",
    exact: true,
  });
  // The published primary hover utility retains the primary fill.
  await button.hover();
  await expect(button).toHaveCSS("background-color", "rgb(89, 98, 240)");
  await page.mouse.down();
  await expect(button).toHaveCSS("background-color", "rgb(89, 98, 240)");
  await page.mouse.up();
  await page.keyboard.press("Shift+Tab");
  await page.keyboard.press("Tab");
  await expect(button).toBeFocused();
  await expect(button).toHaveCSS("background-color", "rgb(89, 98, 240)");
  await expect(button).toHaveCSS("outline-style", "solid");
  await expect(button).toHaveCSS("transition-duration", /^(0s|1e-05s)$/);
  const durations = await page.evaluate(() =>
    ["fast", "normal", "account", "saving", "goal", "shared-goal", "limit"].map(
      (name) =>
        getComputedStyle(document.documentElement)
          .getPropertyValue(`--wk-motion-${name}`)
          .trim(),
    ),
  );
  expect(durations).toEqual(Array(7).fill("0ms"));
  await expect(
    page.getByRole("button", { name: "Disabled", exact: true }),
  ).toBeDisabled();
});

test("font failure retains content and a readable fallback", async ({
  page,
}) => {
  await page.route("**/*.woff*", (route) => route.abort());
  await page.goto("/__design/tokens");
  await page.evaluate(async () => {
    await document.fonts.ready;
  });
  await expect(page.getByTestId("brand-sample")).toContainText(
    "ОП Ёжик • ₽ $ € ₿ Ξ −12,50%",
  );
  await expect(page.getByTestId("brand-sample")).toHaveCSS(
    "font-family",
    /Days One.*sans-serif/,
  );
  await expect(page.getByTestId("amount-BTC")).toHaveText("0.12345678");
  expect(
    await page
      .getByTestId("brand-sample")
      .evaluate((element) => element.getBoundingClientRect().height),
  ).toBeGreaterThan(0);
});

test("critical background is dark before the stylesheet arrives", async ({
  page,
}) => {
  await page.route("**/*.css*", (route) => route.abort());
  await page.goto("/");
  await expect(page.locator("html")).toHaveCSS(
    "background-color",
    "rgb(20, 20, 20)",
  );
});
