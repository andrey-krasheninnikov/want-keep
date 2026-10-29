import { Button as SharedButton } from "@want-foundation/desyst-react";
import type { ComponentProps } from "react";
import { cn } from "../class-names";

export type ButtonProps = Omit<
  ComponentProps<typeof SharedButton>,
  "variant" | "size"
> & {
  variant?: "primary" | "secondary" | "ghost" | "destructive" | "link";
  size?: "sm" | "md" | "lg" | "icon";
};
export function Button({
  variant = "primary",
  size = "md",
  type = "button",
  className,
  ...props
}: ButtonProps) {
  return (
    <SharedButton
      {...props}
      type={type}
      variant={variant === "primary" ? "default" : variant}
      size={size === "md" ? "default" : size}
      className={cn(
        "wk-button",
        `wk-button-${variant}`,
        `wk-button-${size}`,
        className,
      )}
    />
  );
}
