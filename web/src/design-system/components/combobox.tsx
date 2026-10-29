import { Combobox as SharedCombobox } from "@want-foundation/desyst-react";
import type { SelectOption } from "./select";
export type ComboboxProps = {
  id?: string;
  label: string;
  options: readonly SelectOption[];
  value: string | null;
  onValueChange: (value: string | null) => void;
  emptyLabel: string;
  disabled?: boolean;
};
export function Combobox({ options, value, ...props }: ComboboxProps) {
  return (
    <SharedCombobox {...props} value={value ?? ""} options={[...options]} />
  );
}
