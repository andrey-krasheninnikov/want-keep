import { readdir, readFile } from "node:fs/promises";
import { URL } from "node:url";
import { ThemeContract } from "./design/theme-contract.ts";

const assetDirectory = new URL("../dist/assets/", import.meta.url);
const assetNames = await readdir(assetDirectory);
const stylesheetNames = assetNames.filter((name) => name.endsWith(".css"));

if (stylesheetNames.length === 0) {
  throw new Error("The web build did not produce a stylesheet");
}

const stylesheets = await Promise.all(
  stylesheetNames.map((name) =>
    readFile(new URL(name, assetDirectory), "utf8"),
  ),
);
const compiledCss = stylesheets.join("\n").toLowerCase();
const requiredMarkers = [
  "--primary:#5962f0",
  "days one",
  "manrope",
  ".ui-button-default",
  "--color-primary:var(--primary)",
  ".bg-background",
  ".text-accent-readable",
];

for (const marker of requiredMarkers) {
  if (!compiledCss.includes(marker)) {
    throw new Error(`The compiled design theme is missing ${marker}`);
  }
}

const sourceCss = await readFile(
  new URL("../src/design-system/theme.css", import.meta.url),
  "utf8",
);
const theme = new ThemeContract(sourceCss);
for (const pair of theme.matrix()) {
  if (!pair.passes) throw new Error(`Source contrast failed: ${pair.id}`);
}
if (compiledCss.includes("pixelify") || compiledCss.includes("#5f4ef5"))
  throw new Error("Previous theme leaked into the build");
const fontAssets = assetNames.filter((name) => /\.woff2?$/.test(name));
if (
  !fontAssets.some((name) => name.includes("days-one")) ||
  !fontAssets.some((name) => name.includes("manrope"))
)
  throw new Error("Shared local fonts are missing");
const scripts = await Promise.all(
  assetNames
    .filter((name) => name.endsWith(".js"))
    .map((name) => readFile(new URL(name, assetDirectory), "utf8")),
);
if (
  /token-preview|Design foundation|tokens-preview|\/__design\/(tokens|components)|Предельная длина|UISTATE-17|Finance\. Clearer\. Closer\./.test(
    scripts.join("\n"),
  ) ||
  compiledCss.includes(".token-preview") ||
  compiledCss.includes(".component-catalog") ||
  compiledCss.includes(".catalog-composition")
) {
  throw new Error(
    "Development token specimens leaked into the production build",
  );
}
