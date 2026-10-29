export interface ContrastPair {
  id: string;
  foreground: string;
  background: string;
  minimum: number | null;
  purpose: string;
}

const surfaces = [
  "background",
  "card",
  "popover",
  "wk-surface-warm",
  "wk-obsidian",
  "wk-surface-architecture",
];

export const contrastPairs: readonly ContrastPair[] = [
  ...surfaces.flatMap((surface) =>
    [
      "foreground",
      "muted-foreground",
      "accent-readable",
      "success",
      "warning",
      "destructive",
      "wk-sand",
    ].map((text) => ({
      id: `${text}-on-${surface}`,
      foreground: `--${text}`,
      background: `--${surface}`,
      minimum: 4.5,
      purpose: "Normal text / Обычный текст",
    })),
  ),
  ...["primary", "primary-hover", "primary-active"].map((state) => ({
    id: `label-on-${state}`,
    foreground: "--primary-foreground",
    background: `--${state}`,
    minimum: 4.5,
    purpose: "CTA label / Подпись CTA",
  })),
  ...surfaces.flatMap((surface) =>
    ["input", "ring"].map((indicator) => ({
      id: `${indicator}-on-${surface}`,
      foreground: `--${indicator}`,
      background: `--${surface}`,
      minimum: 3,
      purpose: "Control and focus indicators / Индикаторы элементов и фокуса",
    })),
  ),
  {
    id: "brand-large-on-background",
    foreground: "--primary",
    background: "--background",
    minimum: 3,
    purpose:
      "Large brand text only; never small text / Только крупный брендовый текст",
  },
  {
    id: "brand-large-on-obsidian",
    foreground: "--primary",
    background: "--wk-obsidian",
    minimum: 3,
    purpose: "Large wordmark only / Только крупный wordmark",
  },
  {
    id: "keyboard-focus-label",
    foreground: "--wk-focus-foreground",
    background: "--wk-focus-fill",
    minimum: 4.5,
    purpose: "Focused field text / Текст поля с фокусом",
  },
  {
    id: "disabled-label",
    foreground: "--disabled-foreground",
    background: "--disabled",
    minimum: 4.5,
    purpose: "Voluntary readable disabled label / Читаемая неактивная подпись",
  },
  {
    id: "decorative-divider",
    foreground: "--border",
    background: "--card",
    minimum: null,
    purpose:
      "Reserved decorative alias; not rendered / Совместимый декоративный алиас; не отображается",
  },
  {
    id: "disabled-border",
    foreground: "--border",
    background: "--disabled",
    minimum: null,
    purpose:
      "Reserved inactive alias; not rendered / Совместимый неактивный алиас; не отображается",
  },
];
