import { Input as SharedInput } from "@want-foundation/desyst-react";
import type { ComponentProps } from "react";
import { useFieldControl } from "./field-context";
import { cn } from "../class-names";

export type InputProps = ComponentProps<typeof SharedInput> & {
  onValueChange?: (value: string) => void;
};
export function Input({
  className,
  onValueChange,
  onChange,
  ...props
}: InputProps) {
  return (
    <SharedInput
      {...useFieldControl()}
      {...props}
      className={cn("wk-input", "text-foreground", className)}
      onChange={(event) => {
        onValueChange?.(event.currentTarget.value);
        onChange?.(event);
      }}
    />
  );
}
