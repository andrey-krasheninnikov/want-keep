import { Textarea as SharedTextarea } from "@want-foundation/desyst-react";
import type { ComponentProps } from "react";
import { useFieldControl } from "./field-context";
export function Textarea(props: ComponentProps<typeof SharedTextarea>) {
  return (
    <SharedTextarea
      {...useFieldControl()}
      className="wk-input wk-textarea text-foreground"
      {...props}
    />
  );
}
