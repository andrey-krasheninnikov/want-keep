import sharedTokens from "@want-foundation/desyst-foundation/tokens.json" with { type: "json" };
import { contrastPairs } from "../../src/design-system/contrast-pairs.ts";

export class ThemeContract {
  readonly tokens: ReadonlyMap<string, string>;

  constructor(css: string) {
    const root = css.match(/:root\s*\{([^}]+)\}/)?.[1];
    if (!root) throw new Error("Theme has no root token block");
    const tokens = new Map<string, string>(
      Object.entries(sharedTokens.themes.wallpapers.dark).map(
        ([name, value]) => [`--${name}`, value.toLowerCase()],
      ),
    );
    const fonts = (values: string[]) =>
      values
        .map((value) => (value.includes(" ") ? JSON.stringify(value) : value))
        .join(",");
    tokens.set("--want-font-display", fonts(sharedTokens.typography.display));
    tokens.set("--want-font-sans", fonts(sharedTokens.typography.body));
    tokens.set("--radius", `${sharedTokens.geometry.fieldRadiusPx}px`);
    tokens.set("--panel-radius", `${sharedTokens.geometry.panelRadiusPx}px`);
    tokens.set("--media-radius", `${sharedTokens.geometry.mediaRadiusPx}px`);
    tokens.set("--motion-fast", `${sharedTokens.motion.feedbackMs}ms`);
    tokens.set("--motion-transition", `${sharedTokens.motion.transitionMs}ms`);
    const local = new Set<string>();
    for (const match of root.matchAll(/(--[\w-]+)\s*:\s*([^;]+);/g)) {
      if (local.has(match[1])) throw new Error(`Duplicate token ${match[1]}`);
      if (tokens.has(match[1]))
        throw new Error(`Shared token override ${match[1]}`);
      local.add(match[1]);
      tokens.set(match[1], match[2].trim());
    }
    this.tokens = tokens;
  }

  value(name: string): string {
    const visited = new Set<string>();
    let current = name;
    while (!visited.has(current)) {
      visited.add(current);
      const value = this.tokens.get(current);
      if (!value) throw new Error(`Missing theme token ${current}`);
      const alias = /^var\((--[\w-]+)\)$/.exec(value);
      if (!alias) return value;
      current = alias[1];
    }
    throw new Error(`Circular theme token ${name}`);
  }

  static contrast(first: string, second: string): number {
    const luminances = [first, second].map((hex) => {
      const expanded = /^#[\da-f]{3}$/i.test(hex)
        ? "#" + [...hex.slice(1)].map((channel) => channel.repeat(2)).join("")
        : hex;
      if (!/^#[\da-f]{6}$/i.test(expanded))
        throw new Error(`Expected an opaque sRGB color: ${hex}`);
      const [r, g, b] = [1, 3, 5].map((offset) => {
        const channel = parseInt(expanded.slice(offset, offset + 2), 16) / 255;
        return channel <= 0.04045
          ? channel / 12.92
          : ((channel + 0.055) / 1.055) ** 2.4;
      });
      return 0.2126 * r + 0.7152 * g + 0.0722 * b;
    });
    return (Math.max(...luminances) + 0.05) / (Math.min(...luminances) + 0.05);
  }

  matrix() {
    return contrastPairs.map((pair) => {
      const foreground = this.value(pair.foreground);
      const background = this.value(pair.background);
      const ratio = ThemeContract.contrast(foreground, background);
      return {
        ...pair,
        foreground,
        background,
        ratio,
        passes: pair.minimum === null || ratio >= pair.minimum,
      };
    });
  }

  report(): string {
    const rows = this.matrix();
    if (rows.some((row) => !row.passes))
      throw new Error(
        `Contrast failed: ${rows
          .filter((row) => !row.passes)
          .map((row) => row.id)
          .join(", ")}`,
      );
    return [
      "# Want Keep: contrast matrix / Матрица контраста",
      "",
      "Generated from CSS tokens by `node web/scripts/write-contrast-matrix.ts`. Do not edit manually. / Сгенерировано из CSS-токенов; не редактировать вручную.",
      "",
      "Thresholds use unrounded sRGB ratios. Exemptions apply only to the documented roles. / Порог проверяется без округления; исключения относятся только к указанным ролям.",
      "",
      "| Pair / Пара | FG | BG | Ratio | Minimum | Purpose / Назначение |",
      "| --- | --- | --- | --- | --- | --- |",
      ...rows.map(
        (row) =>
          `| ${row.id} | ${row.foreground} | ${row.background} | ${row.ratio.toFixed(3)}:1 | ${row.minimum ?? "exempt"} | ${row.purpose} |`,
      ),
      "",
    ].join("\n");
  }
}
