import { useRef, useState } from "react";
import {
  Form,
  Field,
  FieldLabel,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldSet,
  FieldLegend,
} from "../components/field";
import { Input } from "../components/input";
import { Textarea } from "../components/textarea";
import { MoneyField } from "../components/money-field";
import { DateField } from "../components/date-field";
import { Select } from "../components/select";
import { Combobox } from "../components/combobox";
import { Checkbox } from "../components/checkbox";
import { RadioGroup, RadioItem } from "../components/radio-group";
import { Switch } from "../components/switch";
import { Button } from "../components/button";
import { Alert } from "../components/alert";
import type { CatalogLocale } from "./catalog-locale";

const assets = ["RUB", "USD", "USDT", "USDC", "BTC", "ETH"].map((value) => ({
  value,
  label: value,
}));

export function FormSpecimen({
  locale,
  compact = false,
}: {
  locale: CatalogLocale;
  compact?: boolean;
}) {
  const ru = locale === "ru";
  const [name, setName] = useState("");
  const [amount, setAmount] = useState("0.123456789012345678");
  const [asset, setAsset] = useState<string | null>("ETH");
  const [date, setDate] = useState<{
    value: string | null;
    draft?: string;
    locale: CatalogLocale;
  }>({
    value: "2026-09-07",
    locale,
  });
  const [category, setCategory] = useState<string | null>(null);
  const [notes, setNotes] = useState("");
  const [saving, setSaving] = useState(false);
  const [confirmed, setConfirmed] = useState(false);
  const [submissions, setSubmissions] = useState(0);
  const submitted = useRef(false);
  const amountTooLong = amount.length > 256;
  const options = [
    { value: "food", label: ru ? "Еда" : "Food" },
    { value: "travel", label: ru ? "Транспорт" : "Transport" },
    { value: "health", label: ru ? "Здоровье" : "Health" },
  ];
  return (
    <div className="catalog-stack">
      {!compact && (
        <div className="catalog-heading">
          <p className="catalog-eyebrow">02 / {ru ? "Формы" : "Forms"}</p>
          <h2>
            {ru
              ? "Точный ввод. Спокойная обратная связь."
              : "Precise input. Clear feedback."}
          </h2>
          <p>
            {ru
              ? "Синтетическая форма без отправки данных. Длинные значения доступны целиком."
              : "A synthetic form with no data submission. Long values remain fully available."}
          </p>
        </div>
      )}
      <Form
        className="catalog-form"
        onSubmit={(event) => {
          event.preventDefault();
          if (submitted.current || amountTooLong || !date.value) return;
          submitted.current = true;
          setSaving(true);
          setConfirmed(false);
          setSubmissions((count) => count + 1);
        }}
      >
        <FieldGroup>
          <Field>
            <FieldLabel>{ru ? "Назначение" : "Purpose"}</FieldLabel>
            <Input
              required
              value={name}
              onValueChange={setName}
              placeholder={ru ? "Например, продукты" : "For example, groceries"}
            />
            <FieldDescription>
              {ru
                ? "Понятное вам название записи"
                : "A name you will recognize"}
            </FieldDescription>
            <FieldError />
          </Field>
          <Field invalid={amountTooLong}>
            <FieldLabel>{ru ? "Сумма" : "Amount"}</FieldLabel>
            <MoneyField
              assetLabel={asset ?? ""}
              value={amount}
              onValueChange={setAmount}
              aria-invalid={amountTooLong}
            />
            <FieldDescription>
              {ru
                ? "До 256 символов; исходная точность сохраняется"
                : "Up to 256 characters; original precision is retained"}
            </FieldDescription>
            {amountTooLong && (
              <FieldError>
                {ru
                  ? "Сумма длиннее 256 символов. Ввод сохранён целиком."
                  : "Amount exceeds 256 characters. All input is retained."}
              </FieldError>
            )}
          </Field>
          <Field>
            <FieldLabel>{ru ? "Актив" : "Asset"}</FieldLabel>
            <Select
              label={ru ? "Актив" : "Asset"}
              options={assets}
              value={asset}
              onValueChange={setAsset}
              placeholder={ru ? "Выберите актив" : "Choose an asset"}
            />
          </Field>
          <DateField
            label={ru ? "Дата операции" : "Transaction date"}
            value={date.value}
            draft={
              date.locale === locale || !date.value ? date.draft : undefined
            }
            locale={locale}
            onChange={(next) =>
              setDate({ draft: next.draft, value: next.value ?? null, locale })
            }
          />
          {!compact && (
            <>
              <Field>
                <FieldLabel>{ru ? "Категория" : "Category"}</FieldLabel>
                <Combobox
                  label={ru ? "Категория" : "Category"}
                  emptyLabel={ru ? "Нет совпадений" : "No matches"}
                  options={options}
                  value={category}
                  onValueChange={setCategory}
                />
              </Field>
              <FieldSet>
                <FieldLegend>
                  {ru ? "Принадлежность" : "Allocation"}
                </FieldLegend>
                <RadioGroup
                  aria-label={ru ? "Принадлежность" : "Allocation"}
                  defaultValue="shared"
                >
                  <label className="catalog-check">
                    <RadioItem value="shared" />
                    {ru ? "Совместное" : "Shared"}
                  </label>
                  <label className="catalog-check">
                    <RadioItem value="personal" />
                    {ru ? "Личное" : "Personal"}
                  </label>
                </RadioGroup>
              </FieldSet>
              <label className="catalog-check">
                <Checkbox defaultChecked />
                {ru ? "Прикрепить пояснение" : "Include a note"}
              </label>
              <label className="catalog-check">
                <Switch defaultChecked />
                {ru ? "Напомнить об уточнении" : "Remind me to clarify"}
              </label>
              <Field>
                <FieldLabel htmlFor="sample-note">
                  {ru ? "Комментарий" : "Note"}
                </FieldLabel>
                <Textarea
                  id="sample-note"
                  value={notes}
                  onChange={(event) => setNotes(event.target.value)}
                />
              </Field>
            </>
          )}
          <Button
            type="submit"
            disabled={saving || amountTooLong || !date.value}
            aria-busy={saving}
          >
            {saving
              ? ru
                ? "Сохраняем…"
                : "Saving…"
              : ru
                ? "Проверить форму"
                : "Check form"}
          </Button>
        </FieldGroup>
      </Form>
      <details>
        <summary>
          {ru ? "Введённая сумма полностью" : "Full entered amount"}
        </summary>
        <p className="wk-exact-amount" data-testid="form-exact-amount">
          {amount}
        </p>
      </details>
      {saving && (
        <Alert
          title={ru ? "Ожидается ответ" : "Awaiting a response"}
          announce
          action={
            <Button
              variant="secondary"
              onClick={() => {
                submitted.current = false;
                setSaving(false);
                setConfirmed(true);
              }}
            >
              {ru
                ? "Подтвердить ответ в демонстрации"
                : "Confirm the simulated response"}
            </Button>
          }
        >
          {ru
            ? "Результат ещё не подтверждён. Повторная отправка недоступна."
            : "The outcome is not confirmed. Duplicate submission is unavailable."}
        </Alert>
      )}
      {confirmed && (
        <Alert
          tone="success"
          announce
          title={ru ? "Демонстрация завершена" : "Demo completed"}
        >
          {ru
            ? "Получено явное подтверждение. Финансовая запись не создавалась."
            : "An explicit confirmation was received. No financial entry was created."}
        </Alert>
      )}
      <p className="wk-description" data-testid="submission-count">
        {ru ? "Отправок в демонстрации: " : "Demo submissions: "}
        {submissions}
      </p>
    </div>
  );
}
