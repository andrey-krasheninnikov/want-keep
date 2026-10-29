import {
  Field as SharedField,
  FieldLabel as SharedLabel,
  FieldDescription as SharedDescription,
  FieldError as SharedError,
  FieldGroup,
  FieldSet,
  FieldLegend,
} from "@want-foundation/desyst-react";
import {
  Children,
  cloneElement,
  isValidElement,
  useContext,
  useId,
  type ComponentProps,
} from "react";
import { FieldContext } from "./field-context";
import { cn } from "../class-names";

export { FieldGroup, FieldSet, FieldLegend };
export function Form(props: ComponentProps<"form">) {
  return <form {...props} />;
}
export function Field({
  invalid,
  children,
  className,
  ...props
}: ComponentProps<typeof SharedField> & { invalid?: boolean }) {
  const id = useId();
  const items = Children.toArray(children);
  const descriptions = items.filter(
    (child) =>
      isValidElement<{ children?: unknown }>(child) &&
      (child.type === FieldDescription ||
        (child.type === FieldError && Boolean(child.props.children))),
  );
  const description = descriptions.length
    ? descriptions.map((_, index) => `${id}-description-${index}`).join(" ")
    : undefined;
  return (
    <FieldContext.Provider value={{ id, description }}>
      <SharedField
        {...props}
        data-invalid={invalid || undefined}
        className={cn("wk-field", className)}
      >
        {items.map((child) => {
          const index = descriptions.indexOf(child);
          if (index < 0 || !isValidElement<ComponentProps<"div">>(child))
            return child;
          return cloneElement(child, { id: `${id}-description-${index}` });
        })}
      </SharedField>
    </FieldContext.Provider>
  );
}
export function FieldLabel(props: ComponentProps<typeof SharedLabel>) {
  const field = useContext(FieldContext);
  return <SharedLabel htmlFor={field?.id} className="wk-label" {...props} />;
}
export function FieldDescription(
  props: ComponentProps<typeof SharedDescription>,
) {
  return <SharedDescription className="wk-description" {...props} />;
}
export function FieldError(props: ComponentProps<typeof SharedError>) {
  return <SharedError className="wk-field-error" {...props} />;
}
