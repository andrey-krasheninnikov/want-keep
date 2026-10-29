import {
  DateField as SharedDateField,
  type DateFieldLabels,
} from "@want-foundation/desyst-react";
import { useId, useState } from "react";
import { CalendarDate, type DateLocale } from "./calendar-date";
import { Input } from "./input";

export type DateFieldChange = {
  draft: string;
  value: string | null | undefined;
};
export type DateFieldProps = {
  id?: string;
  value: string | null;
  draft?: string;
  locale: DateLocale;
  label: string;
  disabled?: boolean;
  onChange: (change: DateFieldChange) => void;
};
export function DateField({
  id,
  value,
  draft,
  locale,
  label,
  disabled,
  onChange,
}: DateFieldProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const [touched, setTouched] = useState(false);
  const text = draft ?? CalendarDate.display(value, locale);
  const parsed = CalendarDate.parse(text, locale);
  const invalid = touched && parsed === undefined;
  const ru = locale === "ru";
  const hint = ru ? "ДД.ММ.ГГГГ" : "MM/DD/YYYY";
  const labels: DateFieldLabels = {
    date: label,
    datetime: label,
    calendar: label,
    year: ru ? "Год" : "Year",
    previousYear: ru ? "Предыдущий год" : "Previous year",
    previousMonth: ru ? "Предыдущий месяц" : "Previous month",
    nextYear: ru ? "Следующий год" : "Next year",
    nextMonth: ru ? "Следующий месяц" : "Next month",
    hours: ru ? "Часы" : "Hours",
    minutes: ru ? "Минуты" : "Minutes",
    today: ru ? "Сегодня" : "Today",
    clear: ru ? "Очистить" : "Clear",
    apply: ru ? "Применить" : "Apply",
  };
  return (
    <div className="wk-field">
      <label className="wk-label" htmlFor={inputId}>
        {label}
      </label>
      <div className="wk-date-field">
        <Input
          id={inputId}
          disabled={disabled}
          value={text}
          aria-invalid={invalid}
          aria-describedby={`${inputId}-help`}
          onBlur={() => setTouched(true)}
          onValueChange={(next) =>
            onChange({ draft: next, value: CalendarDate.parse(next, locale) })
          }
        />
        <SharedDateField
          mode="date"
          // Keep invalid drafts distinct from empty and canonical picker values.
          value={parsed === undefined ? `invalid:${text}` : (parsed ?? "")}
          labels={labels}
          locale={locale}
          disabled={disabled}
          ariaLabel={`${ru ? "Открыть календарь" : "Open calendar"}: ${label}`}
          onChange={(next) => {
            setTouched(false);
            onChange({
              value: next || null,
              draft: CalendarDate.display(next || null, locale),
            });
          }}
        />
      </div>
      <p
        id={`${inputId}-help`}
        className={invalid ? "wk-field-error" : "wk-description"}
      >
        {invalid
          ? ru
            ? `Проверьте дату. Формат: ${hint}`
            : `Check the date. Format: ${hint}`
          : hint}
      </p>
    </div>
  );
}
