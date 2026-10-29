import {
  SheetContent as Content,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@want-foundation/desyst-react";
import type { DialogContentProps } from "./dialog";
export { Sheet, SheetTrigger, SheetClose } from "@want-foundation/desyst-react";
export function SheetContent({
  title,
  description,
  children,
  ...props
}: DialogContentProps) {
  return (
    <Content {...props} className="wk-sheet">
      <SheetHeader>
        <SheetTitle>{title}</SheetTitle>
        <SheetDescription>{description}</SheetDescription>
      </SheetHeader>
      {children}
    </Content>
  );
}
