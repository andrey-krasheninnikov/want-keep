import { useState } from "react";
import { Select } from "../components/select";
import { Alert } from "../components/alert";
import { Button } from "../components/button";
import { Input } from "../components/input";
import { Field, FieldLabel } from "../components/field";
import { Empty } from "../components/empty";
import { Skeleton } from "../components/skeleton";
import { Progress } from "../components/progress";
import { ChartLegend, type LegendItem } from "../components/chart-legend";
import { ChatResult } from "../components/chat-result";
import { stateScenarios } from "./state-scenarios";
import type { CatalogLocale } from "./catalog-locale";
export function StatesSpecimen({ locale }: { locale: CatalogLocale }) {
  const ru = locale === "ru";
  const [id, setId] = useState<string | null>("UISTATE-10");
  const [draft, setDraft] = useState("1250.00");
  const [compare, setCompare] = useState(false);
  const [notice, setNotice] = useState("");
  const [hiddenSeries, setHiddenSeries] = useState<string[]>([]);
  const scenario =
    stateScenarios.find((state) => state.id === id) ?? stateScenarios[0]!;
  const legend: LegendItem[] = [
    {
      id: "actual",
      label: ru ? "Факт" : "Actual",
      valueLabel: "12 500 RUB",
      marker: "solid",
      visible: !hiddenSeries.includes("actual"),
    },
    {
      id: "forecast",
      label: ru ? "Прогноз" : "Forecast",
      valueLabel: "18 000 RUB",
      marker: "dashed",
      visible: !hiddenSeries.includes("forecast"),
    },
    {
      id: "reserved",
      label: ru ? "Резерв" : "Reserved",
      valueLabel: "3 000 RUB",
      marker: "dotted",
      visible: !hiddenSeries.includes("reserved"),
    },
  ];
  const blocked = [
    "UISTATE-09",
    "UISTATE-10",
    "UISTATE-12",
    "UISTATE-13",
  ].includes(scenario.id);
  return (
    <div className="catalog-stack">
      <div className="catalog-heading">
        <p className="catalog-eyebrow">06 / {ru ? "Состояния" : "States"}</p>
        <h2>
          {ru
            ? "Что произошло и что делать дальше"
            : "What happened and what comes next"}
        </h2>
        <p>
          {ru
            ? "Все сценарии синтетические. Кнопки переключают локальные образцы."
            : "Every scenario is synthetic. Actions switch local samples."}
        </p>
      </div>
      <Select
        label={ru ? "Сценарий состояния" : "State scenario"}
        placeholder={ru ? "Выберите состояние" : "Choose a state"}
        options={stateScenarios.map((state) => ({
          value: state.id,
          label: `${state.id} · ${state.title[locale]}`,
        }))}
        value={id}
        onValueChange={(next) => {
          setId(next);
          setCompare(false);
          setNotice("");
        }}
      />
      {scenario.id !== "UISTATE-13" && (
        <Field>
          <FieldLabel>{ru ? "Мой черновик" : "My draft"}</FieldLabel>
          <Input value={draft} onValueChange={setDraft} />
        </Field>
      )}
      {scenario.id === "UISTATE-01" && (
        <Skeleton label={ru ? "Получаем записи" : "Loading entries"} />
      )}
      {["UISTATE-03", "UISTATE-04"].includes(scenario.id) ? (
        <Empty
          title={scenario.title[locale]}
          description={scenario.message[locale]}
          action={
            <Button
              variant="secondary"
              onClick={() =>
                setNotice(
                  ru
                    ? "Демонстрация следующего шага открыта"
                    : "The next-step sample is open",
                )
              }
            >
              {scenario.action[locale]}
            </Button>
          }
        />
      ) : (
        <Alert
          title={scenario.title[locale]}
          tone={scenario.tone}
          action={
            <Button
              variant="secondary"
              onClick={() => {
                if (scenario.id === "UISTATE-11") setCompare(true);
                else if (
                  [
                    "UISTATE-01",
                    "UISTATE-02",
                    "UISTATE-06",
                    "UISTATE-07",
                    "UISTATE-09",
                    "UISTATE-10",
                  ].includes(scenario.id)
                ) {
                  setId("UISTATE-16");
                  setNotice(
                    ru
                      ? "Получен подтверждённый ответ той же демонстрационной команды."
                      : "The same demo command returned a confirmed response.",
                  );
                } else
                  setNotice(
                    ru
                      ? "Выбран следующий шаг в демонстрации. Внешних действий нет."
                      : "The next demo step was selected. No external action occurred.",
                  );
              }}
            >
              {scenario.action[locale]}
            </Button>
          }
        >
          {scenario.message[locale]}
        </Alert>
      )}
      {scenario.id !== "UISTATE-13" && (
        <Button
          disabled={blocked}
          onClick={() => {
            setId("UISTATE-09");
            setNotice("");
          }}
        >
          {ru ? "Отправить образец" : "Submit sample"}
        </Button>
      )}
      {compare && (
        <div className="catalog-compare">
          <div>
            <p className="wk-label">
              {ru ? "Участник Б · сохранено" : "Member B · saved"}
            </p>
            <p className="wk-exact-amount">1500.00</p>
          </div>
          <div>
            <p className="wk-label">{ru ? "Мой ввод" : "My input"}</p>
            <p className="wk-exact-amount">{draft}</p>
          </div>
          <Button
            onClick={() => {
              setCompare(false);
              setId("UISTATE-09");
            }}
          >
            {ru
              ? "Применить мой вариант в демонстрации"
              : "Apply my version in the demo"}
          </Button>
        </div>
      )}
      <p className="wk-description" role="status">
        {notice}
      </p>
      <div className="catalog-two-columns">
        <Progress
          label={ru ? "Готовность образца" : "Sample progress"}
          value={60}
          description={
            ru
              ? "60% · значение передано компоненту"
              : "60% · value supplied to the component"
          }
        />
        <Progress
          label={ru ? "Обработка" : "Processing"}
          value={null}
          description={
            ru ? "Объём пока неизвестен" : "Total progress is not yet known"
          }
        />
        <Progress
          label={ru ? "Готово" : "Complete"}
          value={100}
          description={
            ru ? "100% · образец завершён" : "100% · sample completed"
          }
        />
      </div>
      <ChartLegend
        label={ru ? "Легенда аналитики · образец" : "Analytics legend · sample"}
        items={legend}
        onVisibleChange={(seriesId, visible) =>
          setHiddenSeries((items) =>
            visible
              ? items.filter((item) => item !== seriesId)
              : [...items, seriesId],
          )
        }
      />
      <ChatResult
        author={ru ? "Участник А → Want Keep" : "Member A → Want Keep"}
        status={ru ? "Связано с существующей" : "Linked to an existing entry"}
        title={
          ru ? "Чек дополняет запись" : "The receipt adds detail to an entry"
        }
        explanation={
          ru
            ? "Повторного расхода нет. Это отображение переданного подтверждённого результата."
            : "There is no duplicate expense. This renders a supplied confirmed result."
        }
        tone="success"
        actions={
          <Button variant="secondary" asChild>
            <a href="#tables">{ru ? "Посмотреть запись" : "View entry"}</a>
          </Button>
        }
      />
    </div>
  );
}
