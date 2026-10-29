import {
  DialogContent,
  DialogTitle,
  DialogDescription,
} from "@want-foundation/desyst-react";
import type { ComponentProps } from "react";
export {
  Dialog as AlertDialog,
  DialogTrigger as AlertDialogTrigger,
  DialogClose as AlertDialogClose,
} from "@want-foundation/desyst-react";
export function AlertDialogContent({
  title,
  description,
  children,
  ...props
}: Omit<ComponentProps<typeof DialogContent>, "title"> & {
  title: string;
  description: string;
}) {
  return (
    <DialogContent
      {...props}
      role="alertdialog"
      showCloseButton={false}
      className="wk-dialog-composition"
      onPointerDownOutside={(event) => event.preventDefault()}
      onInteractOutside={(event) => event.preventDefault()}
    >
      <DialogTitle>{title}</DialogTitle>
      <DialogDescription>{description}</DialogDescription>
      {children}
    </DialogContent>
  );
}
