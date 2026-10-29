import { Checkbox as SharedCheckbox } from "@want-foundation/desyst-react";
import type { ComponentProps } from "react";
export function Checkbox(
  props: Omit<ComponentProps<typeof SharedCheckbox>, "onCheckedChange"> & {
    onCheckedChange?: (checked: boolean) => void;
  },
) {
  return (
    <SharedCheckbox
      {...props}
      onCheckedChange={(checked) => props.onCheckedChange?.(checked === true)}
    />
  );
}
