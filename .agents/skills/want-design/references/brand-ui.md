# Brand and UI

## Identity and color

Want interfaces put the user's content first. Use the product's approved logo and images without stretching, recoloring, or replacing them with a drawing. Preserve the logo's own colors independently of semantic UI accents.

The current shared theme uses these roles:

| Role | Dark | Light |
| --- | --- | --- |
| Canvas | #141414 | #F6F5F3 |
| Main text | #E8E8E8 | #242321 |
| Card | #1C1C1C | #FFFFFF |
| Popover and feedback | #252525 | #FFFFFF |
| Secondary surface | #252525 | #ECEBE8 |
| Muted text | #ADADAD | #5F5B58 |
| Primary | #5962F0 | #5962F0 |
| Success | #7DF04A | #7DF04A |
| Info | #28D7F5 | #28D7F5 |
| Warning | #FFD43B | #FFD43B |
| Error | #FF526D | #FF526D |

Bright status surfaces use #141414 ink. The `*-text` roles supply readable colored copy on ordinary surfaces. The `*-foreground` roles are ink on the corresponding bright status surfaces. Do not interchange these roles. Ordinary text needs at least 4.5:1 contrast; meaningful icons and control states need 3:1. Pair status color with an icon, label, or explanatory copy.

Use semantic variables and component variants. Do not scatter raw brand hex values through product components or substitute a different palette because a generic design reference suggests it.

## Typography

Days One is the display face at its available weight. Manrope is the body face. Use Noto Sans JP for Japanese and Noto Sans Arabic for Arabic. Serve the supplied font files locally; retain the fallback stacks and licenses.

The scale contains display, h1 through h6, lead, body, small, caption, label, overline, mono, code, code-block, and quote. Use the token scale rather than making each screen's own headings. Match the HTML heading level to document structure, even when the visual variant differs.

Body text and inputs stay at least 16 px; supporting copy and labels stay at least 14 px. Keep reading text around 65 characters per line. Allow text wrapping, localization, and text enlargement. Use monospace and isolated LTR direction for technical values when needed, without forcing the whole interface into LTR.

## Geometry and composition

- Use the 4 px spacing rhythm. Start with content, relationships, and the primary action, then choose a layout that serves them.
- Fields use 12 px corners; panels and media use 24 px; prominent surfaces use 32 px; CTA buttons use capsule corners. Compact feedback islands use 18 px corners.
- Group with fills, spacing, and type. Outline variants remain filled. Do not draw decorative divider rules or wrap every piece of text in a card.
- Let reading surfaces stay opaque. Liquid glass is for suitable action surfaces: 68% neutral tint, 20 px blur, and 120% saturation. Reduced transparency, unsupported filtering, and forced colors use opaque surfaces.
- Use one clear primary action in each decision area. Keep secondary and destructive actions distinct. Destructive labels name what will be removed or changed.
- Preserve the 2 px primary focus outline with a 3 px offset. Focus remains visible without hover. Forced colors use system focus colors.

A marketing page may use an expressive content composition; a form or reading screen needs a quieter one. Do not impose a hero, a fixed number of cards, oversized headings, or decorative motion on every Want project.

## Controls and feedback

Reuse the project's controls before adding a new one. Interactive targets are at least 44 by 44 px, including icon-only actions. Provide names for icon buttons, labels for fields, associated helper/error text, and native keyboard behavior.

Navigation chips and breadcrumbs center their content. Long labels wrap or reflow without losing targets or focus. Preserve logical ordering and logical spacing in RTL; mirror navigation arrows, not artwork or logos.

Toast and snackbar surfaces follow the light or dark theme. Status belongs in the 32 px icon plate, not across the entire message surface. Keep a clear title, optional useful detail, a 44 px close target, and at most one relevant action. Desktop actions sit beside the message; narrow layouts move them below. Errors stay until dismissal by default. Hover and keyboard focus pause timed notifications.

Choose loading feedback that matches the wait: a small Loader for a local operation, Shimmer or Skeleton for a known layout, and Progress when progress is measurable. Supply an accessible label. Do not leave a loading illustration after the state has resolved.

Empty states explain what belongs in the area and how to proceed. Errors explain the observed problem and an available recovery action. Do not imply data was lost, saved, uploaded, or retried without evidence.

## Responsive behavior

Keep the changed flow useful at 320 px as well as desktop. Reduce columns and reposition secondary actions before shrinking type or targets. Prevent page-wide horizontal overflow; use deliberate local scrolling for data or code that needs it. Do not crop labels or remove actions to make a screenshot fit.

Check text enlargement and long translated copy. Preserve keyboard order, headings, native form behavior, focus restoration, and announcements. Consumer code owns localized labels and prepared dates; rendering should not change their text between server and client.
