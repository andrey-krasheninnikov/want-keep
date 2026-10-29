import { Button } from "../components/button";
import { Input } from "../components/input";
import { useState } from "react";

import { BrandSpecimen } from "./BrandSpecimen";

import { contrastPairs } from "../contrast-pairs";
import "./token-preview.css";

const assets = [
  ["RUB", "−123 456 789,01"],
  ["USD", "123,456,789.01"],
  ["USDT", "123.00000001"],
  ["USDC", "0.123456789012345678"],
  ["BTC", "0.12345678"],
  ["ETH", "0.123456789012345678"],
];
const sampleText =
  "ОП Ёжик • ₽ $ € ₿ Ξ −12,50% • 1\u00a0234 • 1\u202f234 • RUB USD USDT USDC BTC ETH";

export function TokenPreview() {
  const [locale, setLocale] = useState<"ru" | "en">("ru");
  const ru = locale === "ru";

  return (
    <main className="token-preview" lang={locale} data-testid="token-preview">
      <header className="token-preview__header">
        <div className="token-preview__logo">
          <img
            src="/brand/logo_512px.svg"
            alt="Want Keep"
            width="455"
            height="512"
          />
        </div>
        <span className="token-preview__wordmark wk-type-brand">Want Keep</span>
        <span className="token-preview__edition">DESIGN FOUNDATION / 01</span>
        <Button variant="secondary" onClick={() => setLocale(ru ? "en" : "ru")}>
          {ru ? "English" : "Русский"}
        </Button>
      </header>
      <BrandSpecimen locale={locale} />
      <section
        aria-labelledby="typography-title"
        className="token-preview__panel"
      >
        <h2 id="typography-title">
          {ru ? "Шрифты и точные суммы" : "Fonts and exact amounts"}
        </h2>
        <p
          className="wk-type-brand token-preview__brand"
          data-testid="brand-sample"
        >
          {sampleText}
        </p>
        <p data-testid="interface-sample">{sampleText}</p>
        <p className="wk-type-numeric" data-testid="tabular-sample">
          1111111111 · 8888888888 · 0000000000
        </p>
        <div className="token-preview__amounts">
          {assets.map(([asset, amount]) => (
            <div key={asset}>
              <span>{asset}</span>
              <p
                className="wk-type-numeric wk-exact-amount"
                data-testid={`amount-${asset}`}
              >
                {amount}
              </p>
            </div>
          ))}
        </div>
        <details>
          <summary>
            {ru
              ? "Предельная длина: 256 символов"
              : "Maximum length: 256 characters"}
          </summary>
          <p
            className="wk-type-numeric wk-exact-amount"
            data-testid="long-amount"
          >
            {"9".repeat(237) + ".123456789012345678"}
          </p>
        </details>
      </section>
      <section aria-labelledby="states-title" className="token-preview__panel">
        <h2 id="states-title">
          {ru ? "Состояния и обратная связь" : "States and feedback"}
        </h2>
        <p>
          {ru
            ? "Это образцы оформления, а не готовые компоненты."
            : "These are style specimens, not the component library."}
        </p>
        <div className="token-preview__states">
          <Button variant="primary">Normal / Hover / Active</Button>
          <Button variant="secondary" aria-pressed="true">
            ✓ Selected
          </Button>
          <Button variant="secondary" disabled>
            Disabled
          </Button>
          <label>
            {ru ? "Название" : "Name"}
            <Input
              placeholder={ru ? "Например, наличные" : "For example, cash"}
            />
          </label>
        </div>
        <ul className="token-preview__statuses">
          <li className="token-preview__success">
            ✓{" "}
            {ru
              ? "Сохранено — пример подтверждения"
              : "Saved — confirmation sample"}
          </li>
          <li className="token-preview__warning">
            !{" "}
            {ru
              ? "Внимание — пример предупреждения"
              : "Attention — warning sample"}
          </li>
          <li className="token-preview__error">
            × {ru ? "Ошибка — пример отказа" : "Error — failure sample"}
          </li>
        </ul>
      </section>
      <section aria-labelledby="colors-title" className="token-preview__panel">
        <h2 id="colors-title">
          {ru ? "Разрешённые цветовые пары" : "Allowed color pairs"}
        </h2>
        <div className="token-preview__pairs">
          {contrastPairs.map((pair) => (
            <article key={pair.id}>
              <div
                className="token-preview__swatch"
                style={{
                  color: `var(${pair.foreground})`,
                  backgroundColor: `var(${pair.background})`,
                }}
              >
                <span aria-hidden="true">Aa 123</span>
              </div>
              <p>{pair.id}</p>
              <small>
                {pair.minimum === null
                  ? ru
                    ? "Декоративное исключение"
                    : "Decorative exemption"
                  : `≥ ${pair.minimum}:1`}
              </small>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
