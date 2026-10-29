import { useState } from "react";
import {
  PlusIcon,
  CheckIcon,
  ArrowUpRightIcon,
  SettingsIcon,
} from "lucide-react";
import { Button } from "../components/button";
import { Badge } from "../components/badge";
import { Avatar } from "../components/avatar";
import { ToggleGroup, ToggleItem } from "../components/toggle-group";
import { Tooltip, TooltipTrigger, TooltipContent } from "../components/tooltip";
import type { CatalogLocale } from "./catalog-locale";
export function ActionsSpecimen({ locale }: { locale: CatalogLocale }) {
  const ru = locale === "ru";
  const [selected, setSelected] = useState(["family"]);
  const [result, setResult] = useState("");
  return (
    <div className="catalog-stack">
      <div className="catalog-heading">
        <p className="catalog-eyebrow">01 / {ru ? "Действия" : "Actions"}</p>
        <h2>{ru ? "Одно ясное следующее действие" : "One clear next step"}</h2>
        <p>
          {ru
            ? "Заливка определяет приоритет. Фокус обозначен контрастной обводкой."
            : "Fill establishes priority. A contrasting outline marks focus."}
        </p>
      </div>
      <div className="catalog-controls">
        <Button
          onClick={() =>
            setResult(
              ru
                ? "Демонстрационное действие выполнено"
                : "Demo action completed",
            )
          }
        >
          <PlusIcon aria-hidden="true" data-icon="inline-start" />
          {ru ? "Добавить запись" : "Add entry"}
        </Button>
        <Button
          variant="secondary"
          onClick={() =>
            setResult(ru ? "Изменения отменены" : "Changes cancelled")
          }
        >
          {ru ? "Отмена" : "Cancel"}
        </Button>
        <Button
          variant="ghost"
          onClick={() =>
            setResult(ru ? "Показаны подробности" : "Details displayed")
          }
        >
          {ru ? "Подробнее" : "Details"}
          <ArrowUpRightIcon aria-hidden="true" data-icon="inline-end" />
        </Button>
        <Button
          variant="destructive"
          onClick={() =>
            setResult(ru ? "Образец действия удаления" : "Demo remove action")
          }
        >
          {ru ? "Удалить" : "Remove"}
        </Button>
        <Button variant="link" asChild>
          <a href="#states">{ru ? "Посмотреть состояния" : "View states"}</a>
        </Button>
        <Tooltip>
          <TooltipTrigger
            aria-label={ru ? "Настройки образца" : "Sample settings"}
            onClick={() =>
              setResult(ru ? "Образец настройки" : "Settings sample")
            }
            asChild
          >
            <Button variant="secondary" size="icon">
              <SettingsIcon aria-hidden="true" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>
            {ru ? "Настройки этого образца" : "Settings for this sample"}
          </TooltipContent>
        </Tooltip>
      </div>
      <div className="catalog-controls">
        <Button disabled>{ru ? "Недоступно" : "Unavailable"}</Button>
        <Button disabled aria-busy="true">
          {ru ? "Сохраняем…" : "Saving…"}
        </Button>
        <Button
          size="sm"
          onClick={() =>
            setResult(
              ru ? "Компактное действие выполнено" : "Compact action completed",
            )
          }
        >
          {ru ? "Компактная" : "Compact"}
        </Button>
        <Button
          size="lg"
          onClick={() =>
            setResult(
              ru ? "Основное действие выполнено" : "Primary action completed",
            )
          }
        >
          {ru ? "Основное действие" : "Primary action"}
        </Button>
      </div>
      <ToggleGroup
        aria-label={ru ? "Представление бюджета" : "Budget view"}
        value={selected}
        onValueChange={setSelected}
      >
        <ToggleItem value="family">{ru ? "Семья" : "Household"}</ToggleItem>
        <ToggleItem value="me">{ru ? "Личное" : "Personal"}</ToggleItem>
      </ToggleGroup>
      <div className="catalog-controls">
        <Badge>
          <CheckIcon size={16} aria-hidden="true" />
          {ru ? "Выбрано" : "Selected"}
        </Badge>
        <Badge tone="success">{ru ? "Подтверждено" : "Confirmed"}</Badge>
        <Badge tone="warning">
          {ru ? "Требует внимания" : "Needs attention"}
        </Badge>
        <Badge tone="error">{ru ? "Ошибка" : "Error"}</Badge>
        <Avatar
          name={ru ? "Профиль с изображением" : "Profile image"}
          initials="WK"
          src="/brand/logo_512px.svg"
        />
      </div>
      <p role="status" className="wk-description">
        {result}
      </p>
    </div>
  );
}
