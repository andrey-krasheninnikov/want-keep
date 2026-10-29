import {
  Select as Root,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@want-foundation/desyst-react";
import { useFieldControl } from "./field-context";

export type SelectOption = { value: string; label: string; disabled?: boolean };
export type SelectProps = {
  id?: string;
  label: string;
  value: string | null;
  options: readonly SelectOption[];
  placeholder: string;
  disabled?: boolean;
  invalid?: boolean;
  describedBy?: string;
  onValueChange: (value: string | null) => void;
};
export function Select({
  id,
  label,
  value,
  options,
  placeholder,
  disabled,
  invalid,
  describedBy,
  onValueChange,
}: SelectProps) {
  const field = useFieldControl();
  return (
    <Root value={value ?? ""} onValueChange={onValueChange} disabled={disabled}>
      <SelectTrigger
        {...field}
        id={id ?? field.id}
        aria-label={label}
        aria-invalid={invalid}
        aria-describedby={describedBy ?? field["aria-describedby"]}
        className="wk-select-trigger"
      >
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent position="popper" className="wk-select-popup">
        {options.map((option) => (
          <SelectItem
            className="wk-select-item"
            key={option.value}
            value={option.value}
            disabled={option.disabled}
          >
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Root>
  );
}
