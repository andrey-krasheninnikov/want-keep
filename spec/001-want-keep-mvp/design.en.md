# Want Keep design contract

[Русский](design.md) · [Screens](screens.en.md) · [Navigation](navigation.en.md)

D-49 updates the visual portion of D-30. REQ-077–REQ-087 and AC-094–AC-104 retain their identifiers. Current migration: task-7.16 / [#97](https://github.com/andrey-krasheninnikov/want-keep/issues/97).

## Platform and character

Desktop web on macOS laptops in Chrome and Arc. Check 1280×720 and 1440×900 CSS px at 100%/200% zoom. Window size and zoom must preserve access to primary actions, forms and critical amounts. Mobile adaptation and a light theme are excluded; browser push remains.

Want Keep uses the complete want-desyst dark theme. Screen structure serves user tasks. The library owns visual primitives; the product owns data and workflows.

## Tokens

Values come from `@want-foundation/desyst-foundation/tokens.json` 0.2.1 and its compiled CSS. Background is `#141414`, card `#1C1C1C`, popover/secondary `#252525`, primary `#5962F0`, foreground `#E8E8E8` and muted text `#ADADAD`. Do not locally replace shared success/info/warning/error roles, geometry or motion. Fields use 12 px corners, panels 24 px and prominent surfaces 32 px; control targets remain at least 44 px.

Load CSS and fonts once. HTML contains `data-theme="dark"` and `dark` before first paint, including with a light OS preference. `web/src/design-system/theme.css` retains only product layout aliases and future event-effect durations. It does not replace shared colors or fonts. Keyboard focus uses the shared contrasting outline; forced-colors and reduced-motion behavior remain intact.

The [contrast matrix](design-contrast.md) derives from the pinned foundation and used roles. Place status text on background/card/popover rather than arbitrary accent surfaces. Normal text requires 4.5:1; large text and meaningful indicators require 3:1. Pair color with text or icons. Separate checks measure rendered component states.

## Typography and logo

Days One serves display; Manrope serves everyday UI and tabular figures. Foundation supplies local fonts.css, fallback and font assets. Remove Pixelify Sans copies and former local font-face definitions. Licenses live in `web/public/licenses/want-desyst/`; the lockfile pins package integrity.

RUB/USD/USDT/USDC/BTC/ETH remain exact strings, including 18 fractional digits and the 256-character limit. Inputs preserve complete invalid drafts. Never convert monetary values through Number or truncate them with an ellipsis.

Keep the [Want Keep logo](assets/logo_512px.svg) byte-identical at its original `455×512` proportions. Do not substitute another product's shared logo.

## Reference sign-in

Source: the user’s image in the conversation; Figma node/exact dimensions were not inspected. This is a composition reference, not a measured design.

Compact mark and Want Keep centered in dark space. Below with a substantial gap, one primary button `Log in with Passkeys` / «Войти с passkey». The block has bounded width accommodating both locales; exported pixels are not copied directly. Subtle language and “Trouble signing in?” / «Не получается войти?» do not compete with the primary action. No bank logos, dashboard cards or password fields on sign-in.

System passkey waiting, cancellation and error use reserved status space; mark/name/CTA do not jump. Error provides retry/help; cancellation is not a malfunction. Recovery remains SCR-002. Financial-event accent animations never play on sign-in.

## Components and accessibility

Use public Radix-based @want-foundation/desyst-react 0.2.1 components with compiled styles and public props. [design-components.en.md](design-components.en.md) documents the migration map, product compositions and form behavior. Features retain authority, APIs, localization and financial state. The shared library makes no business decisions.

## Contextual animations

Animation responds to an event; static text explains the outcome and remains accessible. Neither effect nor color is the sole confirmation. No sound, flashing, strobe or continuously moving decorative backgrounds.

| ID | Event and effect | Constraint |
| --- | --- | --- |
| MOT-01 | Account added to accounting: short pixel rocket launch beside confirmation, 600–1000 ms | Confirmed creation only; not viewing an account or opening a bank product |
| MOT-02 | Confirmed savings top-up: a few sparkles around progress, 600–1000 ms | Principal is not return; forecast and repeat import never trigger it |
| MOT-03 | Goal actually achieved: local pixel confetti, 1000–1600 ms | Once per confirmed achievement event; refund/correction never hidden by celebration |
| MOT-04 | Major joint achievement: soft disco with slowly shifting geometric accents, up to 1800 ms | MOT-03 variant for completed joint goal, no flashing or abrupt brightness change; does not stack with confetti |
| MOT-05 | Limit first exceeded: calm warning-icon appearance, 150–200 ms | Static excess amount and “View plan”; no celebratory reward for overspending |

Decorative effects default on when `prefers-reduced-motion` is absent; each member may disable them in SCR-031. Reduced motion replaces all MOT with static icon/text regardless of preference. Escape ends decoration without cancelling the transaction; effects never capture pointer/focus, obscure amounts/forms or shift layout. Financial numbers immediately show confirmed values without count-up through false amounts.

Trigger is a confirmed typed application event; page read, AI response, optimistic update and quote are not achievement events. Event ID and personal presentation acknowledgement prevent replay in another tab, retry/reconnect or refresh. Historical backfill creates financial facts without a queue of old celebrations. Show one eligible effect at a time; others remain static events. Presentation dedup does not affect personal notification reading. Unknown outcome waits for reconciliation; later correction leaves a clear new result. task-7.15 materializes eligibility/ack contracts; task-7.12 tests actual browsers.

## UX evaluation

For each SCR: question → primary answer → action → explanation → details. Technical fields stay below the primary level; amounts are labelled available/reserved/expected. Errors name the problem/action; partial history explains limits of conclusions. Clear status, familiar language, control, error prevention and progressive disclosure apply to concrete flows. [Nielsen Norman Group](https://www.nngroup.com/articles/ten-usability-heuristics/).

AC-102: without developer hints, both members identify allowance/cash risk, explain an overview amount, enter a receipt and distinguish creation/linking, correct shares, explain a goal reserve, handle reauth/AI waiting, and sign in/recover own access. Protocol records expected answers on synthetic data, actual answers, mistakes, time and browser. Chromium automation replaces neither observation nor actual Chrome/Arc. Full WCAG conformance is not claimed without a separate audit.

## Reproducible specimens

`/__design/tokens` and `/__design/components` remain development/test routes with synthetic data, excluded from the production bundle. See [evidence/issue-97-want-desyst.en.md](evidence/issue-97-want-desyst.en.md) for migration checks and acceptance limits. Earlier task-7.10/task-7.11 evidence preserves the original implementation history and does not establish the current theme.
