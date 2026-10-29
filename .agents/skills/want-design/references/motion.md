# Motion

Motion explains a state or supports the product's visual atmosphere. Keep controls predictable while moving their decoration. Use the shared 120 ms feedback and 180 ms transition tokens for ordinary control changes, with the supplied cubic-bezier(.32,.72,0,1) easing.

Buttons compress on press. Disclosure icons rotate with state. Tabs, accordions, dialogs, sheets, and popovers enter and exit without shifting their placement. Sheet movement follows its physical side. Preserve focus, centering, and native dismiss behavior.

## Glow

Glow is a reusable decorative layer inside a positioned, isolated `glow-host`. It inherits the host's radius and uses the primary hue unless the consumer supplies another color. Keep its layer behind content, non-interactive, and hidden from accessibility APIs.

The current effect has a broad ambient halo and two expanding waves. Each wave runs for 3.6 seconds; the second starts 1.8 seconds apart. Animate transforms and opacity. Keep blur and shadow geometry static. Leave room around the host so waves do not cross controls or clip at a nearby container.

Reduced transparency does not hide Glow or disable its waves. Reduced motion removes movement and leaves a static glow. Forced colors hide the decoration. The `paused` prop stops both waves. Provide a pause control for persistent decorative motion and pause inactive surfaces.

Inspect Glow with the browser's real preferences first. A preview that changes media preferences is additional evidence, not proof that the normal screen shows the effect.

## Feedback and text

Toast and snackbar entry settles over 360 ms with cubic-bezier(.16,1,.3,1); the status icon enters over 420 ms. Exit takes 180 ms. Keep the message readable and preserve close, action, hover/focus pause, and swipe behavior.

AnimatedText offers fade, slide, and blur entry for the new value. Keep only the current text, without stale outgoing content or a timer queue. Let the consumer choose whether the change needs an announcement.

Loaders and shimmers stop looping when paused or when reduced motion is requested. Match the placeholder to the layout so completion does not cause an avoidable jump. Do not announce every decorative frame.

## Optional product effects

The background sphere is opt-in and has an explicit pause/resume control. It stops for hidden documents and reduced motion. The card glow follows its 2.6-second wave without tilting or moving the card. Confetti lasts 800 ms and does not control downloads. The wand preserves selection, editable fields, media, and native pointers; it emits at most three 12 px sparks with .45 peak opacity.

Keep these effects outside administration and reading surfaces. Dispose timers, listeners, observers, and temporary object URLs when the owning view unmounts. Never make decoration a prerequisite for a native link, download, selection, or form action.
