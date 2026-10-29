import {
  PlusIcon,
  ArrowUpRightIcon,
  WalletIcon,
  Music2Icon,
  PiggyBankIcon,
  UsersIcon,
  SettingsIcon,
} from "lucide-react";
import { Surface } from "../components/surface";
import { ActionTile } from "../components/action-tile";
import { Avatar } from "../components/avatar";
import { ListItem } from "../components/list-item";
import { Badge } from "../components/badge";
import { Button } from "../components/button";
import type { CatalogLocale } from "./catalog-locale";
export function CompositionSpecimen({ locale }: { locale: CatalogLocale }) {
  const ru = locale === "ru";
  return (
    <section
      className="catalog-composition"
      aria-label={ru ? "Композиция по референсу" : "Reference composition"}
    >
      <div className="catalog-composition-main">
        <div className="catalog-balance-group">
          <Surface variant="accent" prominent className="catalog-balance">
            <span className="catalog-pixel-star" aria-hidden="true" />
            <p>
              {ru
                ? "Доступно для расходов · образец"
                : "Available to spend · sample"}
            </p>
            <p className="catalog-balance-amount">
              125 000<span>RUB</span>
            </p>
            <Button variant="ghost" size="sm" asChild>
              <a href="#tables">
                {ru ? "Из чего сумма" : "Explain this amount"}
                <ArrowUpRightIcon aria-hidden="true" />
              </a>
            </Button>
          </Surface>
          <div className="catalog-quick-actions">
            <ActionTile icon={<PlusIcon />} href="#forms">
              {ru ? "Добавить запись" : "Add entry"}
            </ActionTile>
            <ActionTile icon={<WalletIcon />} href="#states">
              {ru ? "Проверить учёт" : "Check records"}
            </ActionTile>
          </div>
        </div>
        <div className="catalog-activity">
          <div className="catalog-activity-heading">
            <h3>{ru ? "История" : "History"}</h3>
            <Button variant="secondary" size="sm" asChild>
              <a href="#tables">{ru ? "Все записи" : "View all"}</a>
            </Button>
          </div>
          <Surface className="catalog-activity-row">
            <div>
              <p className="wk-description">
                {ru ? "Общий счёт" : "Shared account"}
              </p>
              <p className="catalog-history-amount">−1 200.00 RUB</p>
              <Badge>{ru ? "Расход" : "Expense"}</Badge>
            </div>
            <ListItem
              leading={<Music2Icon aria-hidden="true" />}
              title={ru ? "Музыка" : "Music"}
              description={ru ? "Месячная подписка" : "Monthly subscription"}
            />
          </Surface>
          <Surface className="catalog-activity-row">
            <div>
              <p className="wk-description">{ru ? "Накопления" : "Savings"}</p>
              <p className="catalog-history-amount" data-income="true">
                +1 800.00 RUB
              </p>
              <Badge tone="success">{ru ? "Начисление" : "Accrual"}</Badge>
            </div>
            <ListItem
              leading={<PiggyBankIcon aria-hidden="true" />}
              title={ru ? "Проценты" : "Interest"}
              description={
                ru ? "Подтверждённый факт · образец" : "Confirmed fact · sample"
              }
            />
          </Surface>
        </div>
      </div>
      <Surface prominent className="catalog-profile">
        <Avatar name={ru ? "Участник А" : "Member A"} initials="A" />
        <h3>{ru ? "Всё важное рядом" : "Everything within reach"}</h3>
        <p className="wk-description">
          {ru ? "Семейное пространство · образец" : "Household space · sample"}
        </p>
        <div className="catalog-profile-people">
          <Avatar name={ru ? "Участник А" : "Member A"} initials="A" />
          <Avatar name={ru ? "Участник Б" : "Member B"} initials="B" />
        </div>
        <ListItem
          leading={<UsersIcon aria-hidden="true" />}
          title={ru ? "Семья" : "Household"}
          description={ru ? "Два участника" : "Two members"}
          trailing={
            <Button
              variant="ghost"
              size="icon"
              aria-label={
                ru ? "Пример семейной навигации" : "Household navigation sample"
              }
              asChild
            >
              <a href="#navigation">
                <ArrowUpRightIcon aria-hidden="true" />
              </a>
            </Button>
          }
        />
        <ListItem
          leading={<SettingsIcon aria-hidden="true" />}
          title={ru ? "Предпочтения" : "Preferences"}
          description={
            ru ? "Ясные состояния и действия" : "Clear states and actions"
          }
          trailing={
            <Button
              variant="ghost"
              size="icon"
              aria-label={ru ? "Примеры состояний" : "State samples"}
              asChild
            >
              <a href="#states">
                <ArrowUpRightIcon aria-hidden="true" />
              </a>
            </Button>
          }
        />
      </Surface>
    </section>
  );
}
