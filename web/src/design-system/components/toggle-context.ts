import { createContext } from "react";
export const ToggleContext = createContext<{
  value: string[];
  first?: string;
  change: (value: string[]) => void;
} | null>(null);
