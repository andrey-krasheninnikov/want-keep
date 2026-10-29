import { createContext, useContext } from "react";
export type ToastMessage = { title: string; description?: string };
export const ToastContext = createContext<{
  add: (message: ToastMessage) => void;
} | null>(null);
export function useToastManager() {
  const manager = useContext(ToastContext);
  if (!manager) throw new Error("Toast provider is missing");
  return manager;
}
