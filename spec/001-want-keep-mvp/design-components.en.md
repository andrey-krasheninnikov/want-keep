# Want Keep components

[Русский](design-components.md) · [Design](design.en.md) · [task-7.16](tasks/task-7.16.md)

## Purpose and ownership

D-49 pins `@want-foundation/desyst-foundation` and `@want-foundation/desyst-react` to 0.2.1. The shared library owns Radix primitives, CSS, fonts, focus and motion. Want Keep uses public exports and compiled assets without copying or scanning library sources. The project-local `want-design` comes from the same package.

The local design-system retains product compositions, safe defaults and public-prop adaptation. Features own APIs, routes, financial calculations, persistence and localization. Base UI, DayPicker, shadcn and the independent calendar are removed. There is one interactive primitive implementation.

## Migration map and interfaces

| Local interface | Shared component / retained product behavior |
| --- | --- |
| Button | Shared Button maps primary/md to default/default. Default type is button; submit is explicit. Links use asChild with one child. Ref, disabled, loading and callbacks use public props. |
| ActionTile | Button composition with an icon and text; href supplies a link. The library does not own routes or financial commands. |
| Input / Textarea / MoneyField | Shared Input/Textarea. onValueChange receives the original string. MoneyField adds the asset and decimal inputMode without Number conversion, truncation or hidden normalization. |
| Form / Field / FieldLabel / FieldDescription / FieldError | Native form and shared Field controls. Product composition connects labels, controls and rendered descriptions/errors with unique IDs. Required validation remains native; callers own monetary and domain validation. |
| FieldGroup / FieldSet / FieldLegend | Direct public exports. |
| Select | Shared Select/Trigger/Value/Content/Item. Options, disabled state, null/placeholder and localized names remain props; Radix owns portals and keyboard behavior. |
| Combobox | Shared controlled Combobox with explicit label/options/emptyLabel; nullable selection maps to an empty string. The catalog does not assign a financial category. |
| DateField / CalendarDate | Shared DateField 0.2.1 selects a day. Local composition preserves manual RU/EN input, empty/invalid/valid drafts and errors. CalendarDate converts calendar strings without shifting the day. No independent calendar or DayPicker remains. |
| Checkbox / RadioGroup / RadioItem / Switch / Tabs | Shared controls; RadioItem names the public RadioGroupItem. Callers own state. |
| ToggleGroup / ToggleItem | Small shared-Button composition for budget-view selection: aria-pressed, one Tab stop, arrows/Home/End and controlled selection. It is not a second library primitive. |
| Dialog / Sheet | Public dialogs, titles, descriptions and localized closeLabel. Product composition supplies content without a separate focus trap or portal. |
| AlertDialog | Confirmation built on shared Dialog: role=alertdialog, no outside dismissal or extra X button; the safe action comes first. |
| Popover / Tooltip | Shared overlays and triggers; local composition connects Popover headings to accessible names. Locale changes update the document language, including portals. |
| Table / TableHead / TableCell | Shared semantic table in a scrollable region. Consumers own prepared values and server filters. |
| Pagination | Two shared Buttons with summary and direction availability. Preserve keyset behavior without inventing a total or final page. |
| Surface / Badge / Alert | Shared Card/Badge/Alert with product content and semantic status roles. Alert announces only when announce is set; color never replaces text. |
| Toast / Toaster | Shared ToastProvider and Toast controls. The local queue holds at most three messages for eight seconds; dismissal, pause and accessibility remain intact. Important outcomes stay on screen. |
| Skeleton / Progress | Shared indicators. Progress receives a prepared value or null and a localized explanation. It neither calculates money nor turns unknown into zero. |
| Avatar / Empty / ListItem / Sidebar | Product content/navigation compositions. Avatar keeps its name and initials after image failure; Sidebar receives prepared links and currentId. |
| ChartLegend / ChatResult | Shared Checkbox/Badge compositions with prepared content. They do not calculate yield, apply proposals or alter principal. |

## States and accessibility

`/__design/components` presents synthetic UISTATE-01–17. Loading, disabled, saving, unknown-result, conflict, session-expired and confirmed keep their meaning. Unknown prevents another effect and retains drafts; conflicts preserve input; expired sessions hide sensitive data. Shared controls make no financial decisions.

Preserve labels, helper/error association, native form semantics, controlled state, refs, keyboard, focus trap/restoration, nested overlays and non-submit defaults. Use the shared contrasting focus outline rather than former inversion. Status text stays on allowed surfaces. Reduced-motion/transparency behavior and cleanup belong to shared implementations and product effects.

Dates support 0001-01-01 through 9999-12-31. Locale changes reformat valid days and retain invalid text for correction. All six assets and long fractional amounts remain strings. Sources, server roles and command actors are independent of the selected view.

## Verification and handoff

Evidence distinguishes web unit/typecheck/lint/build, design-tokens/design-components/access browser suites and docs-check. Chromium alone does not prove Chrome or Arc acceptance. Store screenshots and traces outside the repository; use no personal data or live bank secrets. Both development catalogs stay outside the production bundle.

See [migration results and limits](evidence/issue-97-want-desyst.en.md). Historical task-7.10/task-7.11 evidence remains. New screens, whole-family UX, OpenAI, banking, notification delivery and event effects in #62 stay with their owning tasks.
