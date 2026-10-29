import { Card } from "@want-foundation/desyst-react";
import type { ComponentProps } from "react";
import { cn } from "../class-names";
export function Surface({
  variant = "default",
  prominent = false,
  className,
  ...props
}: ComponentProps<typeof Card> & {
  variant?: "default" | "raised" | "accent";
  prominent?: boolean;
}) {
  return (
    <Card
      {...props}
      className={cn("wk-surface", className)}
      data-variant={variant}
      data-prominent={prominent || undefined}
    />
  );
}
