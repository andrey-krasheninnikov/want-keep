import { useCallback, useRef } from "react";
import { useBeforeUnload, useBlocker } from "react-router";
import {
  AlertDialog,
  AlertDialogContent,
} from "@/design-system/components/alert-dialog";
import { Button } from "@/design-system/components/button";
import { useLocale } from "@/locales/locale";

export function LeaveGuard({ dirty }: { dirty: boolean | (() => boolean) }) {
  const { t } = useLocale();
  const returnFocus = useRef<HTMLElement | null>(null);
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      (typeof dirty === "function" ? dirty() : dirty) &&
      currentLocation.pathname !== nextLocation.pathname,
  );
  useBeforeUnload(
    useCallback(
      (event: BeforeUnloadEvent) => {
        if (typeof dirty === "function" ? dirty() : dirty) {
          event.preventDefault();
          event.returnValue = "";
        }
      },
      [dirty],
    ),
  );
  return (
    <AlertDialog
      open={blocker.state === "blocked"}
      onOpenChange={(open) => {
        if (!open && blocker.state === "blocked") blocker.reset();
      }}
    >
      <AlertDialogContent
        title={t("leaveTitle")}
        description={t("leaveDescription")}
        onOpenAutoFocus={() => {
          returnFocus.current =
            document.activeElement instanceof HTMLElement
              ? document.activeElement
              : null;
        }}
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          returnFocus.current?.focus();
        }}
      >
        <div className="access-actions">
          <Button
            onClick={() => {
              if (blocker.state === "blocked") blocker.reset();
            }}
          >
            {t("stay")}
          </Button>
          <Button
            variant="secondary"
            onClick={() => {
              if (blocker.state === "blocked") blocker.proceed();
            }}
          >
            {t("leave")}
          </Button>
        </div>
      </AlertDialogContent>
    </AlertDialog>
  );
}
