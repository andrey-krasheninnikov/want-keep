import type { ReactNode } from "react";
import { Button, type ButtonProps } from "./button";
import { cn } from "../class-names";

export function ActionTile({
  icon,
  children,
  className,
  href,
  ...props
}: Omit<ButtonProps, "variant" | "size" | "asChild"> & {
  icon: ReactNode;
  href?: string;
}) {
  const content = (
    <>
      <span aria-hidden="true">{icon}</span>
      <span>{children}</span>
    </>
  );
  return (
    <Button
      {...props}
      asChild={Boolean(href)}
      variant="secondary"
      className={cn("wk-action-tile", className)}
    >
      {href ? <a href={href}>{content}</a> : content}
    </Button>
  );
}
