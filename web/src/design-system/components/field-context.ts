import { createContext, useContext } from "react";

export const FieldContext = createContext<{
  id: string;
  description?: string;
} | null>(null);
export function useFieldControl() {
  const field = useContext(FieldContext);
  return { id: field?.id, "aria-describedby": field?.description };
}
