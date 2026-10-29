import {
  ToastProvider,
  Toast,
  ToastTitle,
  ToastDescription,
  ToastClose,
  ToastViewport,
} from "@want-foundation/desyst-react";
import { useRef, useState, type ReactNode } from "react";
import { XIcon } from "lucide-react";
import { Button } from "./button";
import { ToastContext, type ToastMessage } from "./toast-manager";
export function Toaster({
  children,
  closeLabel,
}: {
  children: ReactNode;
  closeLabel: string;
}) {
  const nextId = useRef(0);
  const [messages, setMessages] = useState<(ToastMessage & { id: number })[]>(
    [],
  );
  return (
    <ToastContext.Provider
      value={{
        add(message) {
          const id = ++nextId.current;
          setMessages((current) => [...current, { ...message, id }].slice(-3));
        },
      }}
    >
      <ToastProvider duration={8000}>
        {children}
        {messages.map((message) => (
          <Toast
            key={message.id}
            className="wk-toast"
            onOpenChange={(open) => {
              if (!open)
                setMessages((current) =>
                  current.filter((item) => item.id !== message.id),
                );
            }}
          >
            <div>
              <ToastTitle>{message.title}</ToastTitle>
              <ToastDescription>{message.description}</ToastDescription>
            </div>
            <ToastClose asChild>
              <Button variant="ghost" size="icon" aria-label={closeLabel}>
                <XIcon aria-hidden="true" />
              </Button>
            </ToastClose>
          </Toast>
        ))}
        <ToastViewport className="wk-toast-viewport" />
      </ToastProvider>
    </ToastContext.Provider>
  );
}
