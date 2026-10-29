# React integration

Install matching versions of `@want-foundation/desyst-foundation` and `@want-foundation/desyst-react`. React 19 and React DOM are peers. Import the compiled stylesheet once in the application entry; local fonts are optional:

```tsx
import '@want-foundation/desyst-foundation/styles.css';
import '@want-foundation/desyst-foundation/fonts.css';
```

No library-source scan or copied CSS is needed. Tailwind consumers may import the foundation's `tailwind.css` after Tailwind to register the same semantic utilities for their own code. That integration is optional.

## Choose public components

Use the exported Radix-based primitives for native keyboard behavior, focus, labeling, and dismiss controls. The package also supplies Checkbox, DropdownMenu, Toast, Tooltip, Combobox, Popover, Table, Pagination, FileUpload, and Progress, plus Typography, Prose, Loader, Shimmer, AnimatedText, Glow, and Snackbar.

- Typography separates the visual `variant` from its semantic `as` element. Prose styles native reading content.
- Loader requires `label` and supports spinner, dots, bars, and orbit. Shimmer requires `label` and supports block, text, avatar, and card. Both accept `paused`.
- Snackbar requires `variant`, localized `variantLabel`, `title`, and `closeLabel`. Provide one SnackbarViewport per SnackbarProvider. The consumer owns `open`, `onOpenChange`, duration overrides, and action effects.
- Combobox, Pagination, DateField, and FileUpload use controlled values and consumer-supplied labels. FileUpload selects files; it does not upload them. A null Progress value means indeterminate progress.
- Brand, media, contact, and comment compositions receive product assets, prepared content, links, and callbacks. They do not fetch or persist product records.

## A bounded glow action

```tsx
import { useState } from 'react';
import { Button, Glow, Typography } from '@want-foundation/desyst-react';

export function PreviewAction({ title, label, pauseLabel, resumeLabel, onPreview }: {
  title: string;
  label: string;
  pauseLabel: string;
  resumeLabel: string;
  onPreview: () => void;
}) {
  const [paused, setPaused] = useState(false);
  return <section>
    <Typography variant="h2">{title}</Typography>
    <Button variant="secondary" onClick={() => setPaused(value => !value)}>
      {paused ? resumeLabel : pauseLabel}
    </Button>
    <div className="glow-host" style={{ marginBlock: 32, borderRadius: 24, padding: 24 }}>
      <Glow paused={paused} />
      <Button onClick={onPreview}>{label}</Button>
    </div>
  </section>;
}
```

The example leaves application effects and localized copy in the consumer. Use the product's layout styles and semantic surfaces around it.

## Integration boundaries

Use `data-theme="light"` or `data-theme="dark"` on the document root for an explicit theme, and keep the `dark` class in sync for class-based utilities. Without an explicit theme, foundation styles follow the system preference. ThemeToggle is controlled; the consumer owns persistence and system subscriptions.

Respect logical direction and technical LTR isolation. Pass prepared, localized dates and labels so SSR and hydration agree. Keep routes, APIs, telemetry, advertising, authentication, storage, and localization dictionaries outside the public packages.
