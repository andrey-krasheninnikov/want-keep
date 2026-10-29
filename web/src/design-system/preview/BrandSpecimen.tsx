import "./brand-specimen.css";

export function BrandSpecimen({ locale }: { locale: "ru" | "en" }) {
  const ru = locale === "ru";
  return (
    <section className="brand-specimen" aria-labelledby="preview-title">
      <div className="brand-specimen__intro">
        <h1 id="preview-title">
          {ru ? "Токены и типографика" : "Tokens and typography"}
        </h1>
        <p>{ru ? "Общая дизайн-система Want" : "Shared Want design system"}</p>
      </div>
      <div className="brand-specimen__scene">
        <svg
          className="brand-specimen__architecture"
          viewBox="0 0 400 520"
          aria-hidden="true"
        >
          <path
            className="brand-specimen__arch-back"
            d="M20 520V270C20 150 155 165 210 30C265 165 400 150 400 270V520Z"
          />
          <path
            className="brand-specimen__arch-light"
            d="M90 520V300C90 205 174 204 215 104C256 204 340 205 340 300V520Z"
          />
          <path
            className="brand-specimen__arch-front"
            d="M106 520V302C106 211 184 214 223 124C262 214 350 211 350 302V520Z"
          />
        </svg>
        <span className="brand-specimen__corner">
          MONEY
          <br />
          UNDER
          <br />
          CONTROL
        </span>
        <div className="brand-specimen__identity">
          <img src="/brand/logo_512px.svg" alt="" width="455" height="512" />
          <p className="brand-specimen__name wk-type-brand">
            WANT <span>KEEP</span>
          </p>
          <p className="brand-specimen__tagline">FINANCE MADE SIMPLE</p>
        </div>
        <span className="brand-specimen__footnote">
          SAME GOALS.
          <br />
          MORE FREEDOM.
        </span>
      </div>
      <p className="brand-specimen__caption">
        {ru
          ? "Образцы для разработки · Общие цвета и шрифты want-desyst. Финансовых операций здесь нет."
          : "Development specimens · Shared want-desyst colors and fonts. No financial operations here."}
      </p>
    </section>
  );
}
