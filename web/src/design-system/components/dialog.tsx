import {
  DialogContent as Content,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@want-foundation/desyst-react";
import type { ComponentProps } from "react";
export {
  Dialog,
  DialogTrigger,
  DialogClose,
} from "@want-foundation/desyst-react";
export type DialogContentProps = Omit<
  ComponentProps<typeof Content>,
  "title"
> & { title: string; description: string; closeLabel: string };
export function DialogContent({
  title,
  description,
  children,
  ...props
}: DialogContentProps) {
  return (
    <Content {...props} className="wk-dialog-composition">
      <DialogHeader>
        <DialogTitle className="wk-dialog-title">{title}</DialogTitle>
        <DialogDescription>{description}</DialogDescription>
      </DialogHeader>
      {children}
    </Content>
  );
}
