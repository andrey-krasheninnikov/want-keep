---
name: want-design
description: Design, build, and refine interfaces for Want products using the shared brand, design tokens, composition patterns, motion, accessibility, and product voice. Use for Want screens, components, UI copy, and design reviews; keep unrelated brands outside this skill.
---

# Want design

Build interfaces that belong to the Want product family. Keep the approved identity of the specific product, its users, and the task in view. A request for a component does not authorize a page redesign or a new dependency.

## Start from the product

- Identify the product, audience, main action, content, and rendering framework from the project before choosing a composition.
- Preserve approved product logos, artwork, routes, terminology, and localized copy. Do not reuse the Want Wallpapers logo as another product's identity.
- Read the project's own instructions and theme. Explicit product decisions take precedence over shared defaults.
- Reuse the installed Want foundation and framework adapter. For a framework without an adapter, translate the same tokens and behaviors into its native controls; do not invent a new shared adapter as part of a screen task.

## Read the relevant reference

- For appearance, hierarchy, layout, controls, and responsive behavior, read [Brand and UI](references/brand-ui.md).
- For text, labels, errors, empty states, and notifications, read [Voice](references/voice.md).
- For transitions, loading, Glow, and decorative effects, read [Motion](references/motion.md).
- For React implementations, read [React integration](references/react.md) and use public package exports.

Exact values come from the foundation tokens for the product's installed version. The distributed skill includes `references/tokens.json`. In this repository, the canonical source is `packages/foundation/tokens.json`; the build copies it into the published skill. Written defaults describe the current Want theme and do not replace explicit product theme values.

## Shared decisions

Use Days One for display and Manrope for body text, with the supplied locale fonts. Keep readable copy, generous control targets, filled neutral surfaces, and a clear primary action. The primary accent is #5962F0. Success #7DF04A, info #28D7F5, warning #FFD43B, and error #FF526D remain identical in light and dark themes. Use their readable text roles for inline copy, not the bright accent on a light canvas.

Keep UI copy calm, concrete, and useful. Describe the state and the next available action. Do not promise success before it is confirmed, blame the user, or substitute sales language for an explanation.

Public components receive content, labels, state, and callbacks. Product data, network calls, persistence, localization dictionaries, analytics, advertising, and routes stay in the consumer.

## Finish at the right level

Check the changed flow in the product's real theme and accessibility settings. For visual work, inspect a narrow layout and a desktop layout. Include keyboard focus, relevant empty/loading/error states, and RTL when the product supports it. Check contrast when color roles change.

Choose checks proportionate to the task and reuse results whose inputs have not changed. A build is not proof of rendered behavior. State what was checked and what remains unverified. This skill does not authorize publication, deployment, or unrelated changes.
