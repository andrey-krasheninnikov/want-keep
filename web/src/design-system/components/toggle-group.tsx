import {
  Children,
  isValidElement,
  useContext,
  type ComponentProps,
} from "react";
import { Button } from "./button";
import { ToggleContext } from "./toggle-context";
export function ToggleGroup({
  value,
  onValueChange,
  children,
  ...props
}: ComponentProps<"div"> & {
  value: string[];
  onValueChange: (value: string[]) => void;
}) {
  const first = Children.toArray(children).find(
    isValidElement<{ value: string }>,
  );
  return (
    <ToggleContext.Provider
      value={{ value, first: first?.props.value, change: onValueChange }}
    >
      <div
        {...props}
        role="group"
        className="wk-toggle-group"
        onKeyDown={(event) => {
          const keys = ["ArrowLeft", "ArrowRight", "Home", "End"];
          if (!keys.includes(event.key)) return;
          const buttons = Array.from(
            event.currentTarget.querySelectorAll<HTMLButtonElement>(
              "button:not(:disabled)",
            ),
          );
          const current = buttons.indexOf(event.target as HTMLButtonElement);
          if (current < 0 || !buttons.length) return;
          const direction = event.key === "ArrowLeft" ? -1 : 1;
          const next =
            event.key === "Home"
              ? 0
              : event.key === "End"
                ? buttons.length - 1
                : (current + direction + buttons.length) % buttons.length;
          event.preventDefault();
          buttons[next]?.focus();
        }}
      >
        {children}
      </div>
    </ToggleContext.Provider>
  );
}
export function ToggleItem({
  value,
  ...props
}: Omit<ComponentProps<typeof Button>, "value"> & { value: string }) {
  const group = useContext(ToggleContext);
  if (!group) throw new Error("Toggle group is missing");
  const selected = group.value.includes(value);
  return (
    <Button
      {...props}
      variant={selected ? "secondary" : "ghost"}
      aria-pressed={selected}
      tabIndex={(group.value[0] ?? group.first) === value ? 0 : -1}
      onClick={() => group.change(selected ? [] : [value])}
    />
  );
}
