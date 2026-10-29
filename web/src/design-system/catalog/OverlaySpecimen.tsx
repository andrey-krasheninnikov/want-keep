import { useState } from "react";
import { Button } from "../components/button";
import { Dialog, DialogTrigger, DialogContent } from "../components/dialog";
import {
  AlertDialog,
  AlertDialogTrigger,
  AlertDialogContent,
  AlertDialogClose,
} from "../components/alert-dialog";
import { Sheet, SheetTrigger, SheetContent } from "../components/sheet";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverTitle,
  PopoverDescription,
} from "../components/popover";
import { useToastManager } from "../components/toast-manager";
import { Alert } from "../components/alert";
import { FormSpecimen } from "./FormSpecimen";
import type { CatalogLocale } from "./catalog-locale";
export function OverlaySpecimen({ locale }: { locale: CatalogLocale }) {
  const ru = locale === "ru";
  const [cancelled, setCancelled] = useState(false);
  const toast = useToastManager();
  return (
    <div className="catalog-stack">
      <div className="catalog-heading">
        <p className="catalog-eyebrow">05 / {ru ? "Слои" : "Overlays"}</p>
        <h2>
          {ru
            ? "Детали без потери контекста"
            : "Details without losing context"}
        </h2>
      </div>
      <div className="catalog-controls">
        <Dialog>
          <DialogTrigger asChild>
            <Button>{ru ? "Открыть форму" : "Open form"}</Button>
          </DialogTrigger>
          <DialogContent
            title={ru ? "Новая запись · образец" : "New entry · sample"}
            description={
              ru
                ? "Форма с календарём и выбором актива. Данные остаются в памяти."
                : "A form with calendar and asset selection. Data stays in memory."
            }
            closeLabel={ru ? "Закрыть форму" : "Close form"}
          >
            <FormSpecimen locale={locale} compact />
          </DialogContent>
        </Dialog>
        <Sheet>
          <SheetTrigger asChild>
            <Button variant="secondary">
              {ru ? "Открыть детали" : "Open details"}
            </Button>
          </SheetTrigger>
          <SheetContent
            title={ru ? "Как получена сумма" : "How the amount was calculated"}
            description={
              ru
                ? "Пример боковой панели, без запроса данных."
                : "Side panel sample, with no data request."
            }
            closeLabel={ru ? "Закрыть детали" : "Close details"}
          >
            <p className="wk-exact-amount">0.123456789012345678 ETH</p>
            <Alert
              title={ru ? "История неполная" : "History is incomplete"}
              tone="warning"
            >
              {ru
                ? "Источник доступен с 1 сентября. Более ранние операции не входят в итог."
                : "The source is available from September 1. Earlier entries are outside this result."}
            </Alert>
          </SheetContent>
        </Sheet>
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="destructive">
              {ru ? "Отменить образец" : "Cancel sample"}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent
            title={
              ru
                ? "Отменить демонстрационную запись?"
                : "Cancel the demo entry?"
            }
            description={
              ru
                ? "Это образец подтверждения; финансовая история не изменяется."
                : "This is a confirmation sample; financial history is unchanged."
            }
          >
            <div className="catalog-controls">
              <AlertDialogClose asChild>
                <Button variant="secondary">{ru ? "Сохранить" : "Keep"}</Button>
              </AlertDialogClose>
              <AlertDialogClose onClick={() => setCancelled(true)} asChild>
                <Button variant="destructive">
                  {ru ? "Да, отменить" : "Yes, cancel"}
                </Button>
              </AlertDialogClose>
            </div>
          </AlertDialogContent>
        </AlertDialog>
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="secondary">
              {ru ? "Быстрая подсказка" : "Quick explanation"}
            </Button>
          </PopoverTrigger>
          <PopoverContent>
            <PopoverTitle className="wk-label">
              {ru ? "Доступность средств" : "Availability of funds"}
            </PopoverTitle>
            <PopoverDescription className="wk-description">
              {ru
                ? "Факт, прогноз и резерв показываются раздельно."
                : "Actual, forecast and reserved amounts are shown separately."}
            </PopoverDescription>
          </PopoverContent>
        </Popover>
        <Button
          variant="secondary"
          onClick={() =>
            toast.add({
              title: ru ? "Скопировано · образец" : "Copied · sample",
              description: ru
                ? "Краткая обратная связь. Важный результат остаётся на экране."
                : "Brief feedback. Important results remain on screen.",
            })
          }
        >
          {ru ? "Показать toast" : "Show toast"}
        </Button>
      </div>
      {cancelled && (
        <Alert announce title={ru ? "Образец отменён" : "Sample cancelled"}>
          {ru
            ? "Подтверждение остаётся на экране после закрытия диалога."
            : "Confirmation remains on screen after the dialog closes."}
        </Alert>
      )}
    </div>
  );
}
