import { readFile } from "node:fs/promises";
import { createRequire } from "node:module";
import sharedTokens from "@want-foundation/desyst-foundation/tokens.json" with { type: "json" };
import { describe, expect, it } from "vitest";
import { ThemeContract } from "../scripts/design/theme-contract";
import { contrastPairs } from "@/design-system/contrast-pairs";

const web = new URL("../", import.meta.url);
const css = await readFile(new URL("src/design-system/theme.css", web), "utf8");
const theme = new ThemeContract(css);
const require = createRequire(import.meta.url);

describe("design-tokens", () => {
  it("uses the published dark theme without product palette overrides", () => {
    for (const [name, value] of Object.entries(
      sharedTokens.themes.wallpapers.dark,
    ))
      expect(theme.value(`--${name}`)).toBe(value.toLowerCase());
    expect(theme.value("--radius")).toBe(
      `${sharedTokens.geometry.fieldRadiusPx}px`,
    );
    expect(theme.value("--wk-radius-panel")).toBe(
      `${sharedTokens.geometry.panelRadiusPx}px`,
    );
    expect(css).not.toMatch(/#[0-9a-f]/i);
    expect(
      () =>
        new ThemeContract(
          css.replace(":root {", ":root { --primary: #000000;"),
        ),
    ).toThrow("Shared token override");
  });
  it("uses shared motion timings and keeps reduced-motion fallbacks", () => {
    expect(theme.value("--wk-motion-fast")).toBe(
      `${sharedTokens.motion.feedbackMs}ms`,
    );
    expect(theme.value("--wk-motion-normal")).toBe(
      `${sharedTokens.motion.transitionMs}ms`,
    );
    expect(css).toContain("prefers-reduced-motion: reduce");
    expect(css).toContain("--wk-motion-goal: 0ms");
  });
  it("checks color roles and the reproducible contrast report", async () => {
    expect(new Set(contrastPairs.map((pair) => pair.id)).size).toBe(
      contrastPairs.length,
    );
    expect(theme.matrix().filter((row) => !row.passes)).toEqual([]);
    expect(
      await readFile(
        new URL("../spec/001-want-keep-mvp/design-contrast.md", web),
        "utf8",
      ),
    ).toBe(theme.report());
    expect(ThemeContract.contrast("#000000", "#ffffff")).toBe(21);
    expect(ThemeContract.contrast("#000", "#fff")).toBe(21);
    expect(() => theme.value("--unknown")).toThrow();
  });
  it("loads the shared local fonts once, including Days One and Manrope", async () => {
    const global = await readFile(
      new URL("src/styles/global.css", web),
      "utf8",
    );
    const typography = await readFile(
      new URL("src/design-system/typography.css", web),
      "utf8",
    );
    const fonts = await readFile(
      require.resolve("@want-foundation/desyst-foundation/fonts.css"),
      "utf8",
    );
    expect(global.match(/desyst-foundation\/fonts\.css/g)).toHaveLength(1);
    expect(global.match(/desyst-foundation\/styles\.css/g)).toHaveLength(1);
    expect(fonts).toContain("Days One");
    expect(fonts).toContain("Manrope");
    expect(fonts).not.toMatch(/https?:\/\//);
    expect(typography).not.toContain("@font-face");
    expect(typography).not.toContain("Pixelify");
    expect(typography).toContain("lining-nums tabular-nums");
    expect(theme.value("--wk-font-brand")).toContain("Days One");
    expect(theme.value("--wk-font-interface")).toContain("Manrope");
  });
  it("keeps the product logo and applies dark mode before the entry script", async () => {
    const logo = await readFile(new URL("public/brand/logo_512px.svg", web));
    expect(
      logo.equals(
        await readFile(
          new URL("../spec/001-want-keep-mvp/assets/logo_512px.svg", web),
        ),
      ),
    ).toBe(true);
    const html = await readFile(new URL("index.html", web), "utf8");
    expect(html).toContain('data-theme="dark" class="dark"');
    expect(html).toContain(`background: ${theme.value("--background")}`);
    expect(html).toContain(
      `name="theme-color" content="${theme.value("--background")}"`,
    );
    expect(html).toContain('name="color-scheme" content="only dark"');
  });
  it("pins registry packages without private tarball paths", async () => {
    const lock = await readFile(new URL("package-lock.json", web), "utf8");
    expect(lock).not.toContain("/private/tmp");
    expect(lock).not.toContain('"@base-ui/react"');
  });
});
