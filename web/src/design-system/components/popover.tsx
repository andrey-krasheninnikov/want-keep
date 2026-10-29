import {
  Popover as Root,
  PopoverContent as Content,
} from "@want-foundation/desyst-react";
import { useContext, useId, type ComponentProps } from "react";
import { PopoverContext } from "./popover-context";
export { PopoverTrigger, PopoverClose } from "@want-foundation/desyst-react";
export function Popover(props: ComponentProps<typeof Root>) {
  const id = useId();
  return (
    <PopoverContext.Provider value={id}>
      <Root {...props} />
    </PopoverContext.Provider>
  );
}
export function PopoverTitle(props: ComponentProps<"h3">) {
  return <h3 id={useContext(PopoverContext)} {...props} />;
}
export function PopoverDescription(props: ComponentProps<"p">) {
  return <p {...props} />;
}
export function PopoverContent(props: ComponentProps<typeof Content>) {
  return (
    <Content
      aria-labelledby={useContext(PopoverContext)}
      className="wk-popover"
      {...props}
    />
  );
}
