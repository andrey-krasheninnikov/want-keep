import { expect, test } from "@playwright/test";
import { ThemeContract } from "../scripts/design/theme-contract";

test.beforeEach(async ({ page }) => {
  await page.goto("/__design/components");
});

for (const viewport of [
  { width: 1280, height: 720 },
  { width: 1440, height: 900 },
  { width: 640, height: 360 },
  { width: 720, height: 450 },
]) {
  test(`composition and form access at ${viewport.width} CSS pixels`, async ({
    page,
  }, testInfo) => {
    await page.setViewportSize(viewport);
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    await page.getByRole("link", { name: "Формы", exact: true }).click();
    await expect(page.getByLabel("Назначение", { exact: true })).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    const panel = page.locator(".catalog-balance");
    await expect(panel).toHaveCSS("border-radius", "32px");
    await expect(panel).toHaveCSS("border-width", "0px");
    await expect(page.getByLabel("Сумма", { exact: true })).toHaveCSS(
      "border-width",
      "0px",
    );
    await page
      .getByRole("button", { name: "Открыть форму", exact: true })
      .click();
    const dialog = page.getByRole("dialog", { name: "Новая запись · образец" });
    await expect(dialog).toBeVisible();
    await dialog
      .getByRole("button", { name: "Проверить форму", exact: true })
      .scrollIntoViewIfNeeded();
    expect(
      await dialog.evaluate(
        (element) => element.scrollWidth <= element.clientWidth,
      ),
    ).toBe(true);
    await page.keyboard.press("Escape");
    await expect(dialog).not.toBeVisible();
    await expect(
      page.getByRole("button", { name: "Открыть форму", exact: true }),
    ).toBeFocused();
    await page.screenshot({
      path: testInfo.outputPath(`components-${viewport.width}.png`),
      fullPage: false,
    });
    expect(errors).toEqual([]);
  });
}

test("precise string entry, error association and duplicate submit protection", async ({
  page,
}) => {
  const amount = page.getByLabel("Сумма", { exact: true });
  for (const value of [
    "-",
    "0.",
    "1,25",
    "0.123456789012345678",
    "9".repeat(256),
    "9".repeat(257),
  ]) {
    await amount.fill(value);
    await expect(amount).toHaveValue(value);
  }
  await expect(amount).toHaveAttribute("aria-invalid", "true");
  await expect(
    page.getByText("Сумма длиннее 256 символов. Ввод сохранён целиком."),
  ).toBeVisible();
  await page.getByText("Введённая сумма полностью", { exact: true }).click();
  await expect(page.getByTestId("form-exact-amount")).toHaveText(
    "9".repeat(257),
  );
  await amount.fill("0.123456789012345678");
  await page
    .getByLabel("Назначение", { exact: true })
    .fill("Синтетическая запись");
  await page
    .getByRole("button", { name: "Проверить форму", exact: true })
    .click();
  await expect(
    page.locator("#forms").getByRole("button", { name: "Сохраняем…" }),
  ).toBeDisabled();
  await page.getByLabel("Назначение", { exact: true }).press("Enter");
  await expect(page.getByTestId("submission-count")).toHaveText(
    "Отправок в демонстрации: 1",
  );
  await expect(
    page.getByText("Демонстрация завершена", { exact: true }),
  ).not.toBeVisible();
  await page
    .getByRole("button", { name: "Подтвердить ответ в демонстрации" })
    .click();
  await expect(
    page.getByText("Демонстрация завершена", { exact: true }),
  ).toBeVisible();
});

test("calendar validation and selection inside a dialog restore focus", async ({
  page,
}) => {
  await page
    .getByRole("button", { name: "Открыть форму", exact: true })
    .click();
  const dialog = page.getByRole("dialog", { name: "Новая запись · образец" });
  const input = dialog.getByRole("textbox", {
    name: "Дата операции",
    exact: true,
  });
  await input.fill("31.02.2026");
  await input.press("Tab");
  await expect(input).toHaveValue("31.02.2026");
  await expect(input).toHaveAttribute("aria-invalid", "true");
  await input.fill("29.02.2024");
  await input.press("Tab");
  await expect(input).toHaveAttribute("aria-invalid", "false");
  const trigger = dialog.getByRole("button", {
    name: "Открыть календарь: Дата операции",
  });
  await trigger.click();
  const calendar = page.locator('[data-slot="date-popover"]');
  await expect(calendar).toBeVisible();
  await expect(calendar.locator("button:focus")).toHaveCount(1);
  await page.keyboard.press("ArrowLeft");
  await page.keyboard.press("Enter");
  await expect(input).toHaveValue("28.02.2024");
  await expect(trigger).toBeFocused();
  await dialog.getByRole("combobox", { name: "Актив" }).click();
  await page.getByRole("option", { name: "BTC", exact: true }).click();
  await expect(dialog.getByRole("combobox", { name: "Актив" })).toContainText(
    "BTC",
  );
  await page.keyboard.press("Escape");
  await expect(dialog).not.toBeVisible();
  await expect(
    page.getByRole("button", { name: "Открыть форму", exact: true }),
  ).toBeFocused();
});

for (const timezoneId of ["Pacific/Honolulu", "Asia/Tokyo", "Europe/Berlin"]) {
  test(`date keeps its day in ${timezoneId}`, async ({ browser }) => {
    const context = await browser.newContext({ timezoneId });
    const page = await context.newPage();
    await page.goto("http://127.0.0.1:4180/__design/components");
    await page
      .getByRole("textbox", { name: "Дата операции", exact: true })
      .fill("29.03.2026");
    await page
      .getByRole("button", { name: "Открыть календарь: Дата операции" })
      .click();
    await expect(
      page.locator(
        '[data-slot="date-popover"] [data-day][aria-pressed="true"]',
      ),
    ).toHaveText("29");
    await page.keyboard.press("Enter");
    await expect(
      page.getByRole("textbox", { name: "Дата операции", exact: true }),
    ).toHaveValue("29.03.2026");
    await context.close();
  });
}

test("combobox, table filters, pagination, tabs and toggles respond to keyboard", async ({
  page,
}) => {
  const category = page.getByRole("combobox", {
    name: "Категория",
    exact: true,
  });
  await category.fill("несуществующая");
  await expect(page.getByText("Нет совпадений", { exact: true })).toBeVisible();
  await category.fill("Транс");
  await category.press("ArrowDown");
  await category.press("Enter");
  await expect(category).toHaveValue("Транспорт");
  await page.getByRole("button", { name: "Далее", exact: true }).click();
  await expect(page.getByTestId("table-ETH")).toHaveText(
    "0.123456789012345678",
  );
  await page.getByLabel("Найти запись или актив").fill("missing");
  await expect(
    page.getByRole("heading", { name: "Нет совпадений", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Сбросить фильтр" }).click();
  await expect(page.getByTestId("table-RUB")).toHaveText("125000.00");
  const tab = page.getByRole("tab", { name: "Семейное", exact: true });
  await tab.focus();
  await tab.press("ArrowRight");
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("tab", { name: "Личное", exact: true }),
  ).toHaveAttribute("aria-selected", "true");
  const checkbox = page.getByRole("checkbox", {
    name: "Прикрепить пояснение",
    exact: true,
  });
  await checkbox.focus();
  await checkbox.press("Space");
  await expect(checkbox).not.toBeChecked();
});

test("unknown outcomes and conflicts preserve drafts, all 17 states explain a next step", async ({
  page,
}) => {
  const select = page.getByRole("combobox", { name: "Сценарий состояния" });
  await page.getByLabel("Мой черновик").fill("999.001");
  await expect(
    page.getByRole("button", { name: "Отправить образец", exact: true }),
  ).toBeDisabled();
  await page
    .getByRole("button", { name: "Проверить исходную команду", exact: true })
    .click();
  await expect(select).toContainText("UISTATE-16");
  await expect(page.getByLabel("Мой черновик")).toHaveValue("999.001");
  await select.click();
  await page
    .getByRole("option", { name: "UISTATE-11 · Конфликт версии", exact: true })
    .click();
  await page.getByRole("button", { name: "Сравнить изменения" }).click();
  await expect(page.locator(".catalog-compare")).toContainText("999.001");
  await expect(page.getByLabel("Мой черновик")).toHaveValue("999.001");
  for (let index = 1; index <= 17; index++) {
    const id = `UISTATE-${String(index).padStart(2, "0")}`;
    await select.click();
    await page.getByRole("option", { name: new RegExp(`^${id} ·`) }).click();
    await expect(select).toContainText(id);
    if (index === 13)
      await expect(page.getByLabel("Мой черновик")).not.toBeVisible();
    else await expect(page.getByLabel("Мой черновик")).toHaveValue("999.001");
  }
});

test("focus, errors, selected state, reduced motion and font failure retain readable content", async ({
  page,
}) => {
  await page.emulateMedia({ colorScheme: "light", reducedMotion: "reduce" });
  const input = page.getByLabel("Сумма", { exact: true });
  await input.fill("9".repeat(257));
  await input.focus();
  await expect(input).toHaveAttribute("aria-invalid", "true");
  await expect(input).toHaveCSS("outline-width", "2px");
  await expect(input).toHaveCSS("color", "rgb(232, 232, 232)");
  await expect(input).toHaveCSS("outline-style", "solid");
  await expect(input).toHaveCSS(
    "transition-duration",
    /^(0s|1e-05s)(, (0s|1e-05s))*$/,
  );
  const toggle = page.getByRole("button", { name: "Семья", exact: true });
  await toggle.focus();
  await toggle.hover();
  await expect(toggle).toHaveCSS("background-color", "rgb(28, 28, 28)");
  await page.route("**/*.woff*", (route) => route.abort());
  await page.reload();
  await page.getByRole("button", { name: "English", exact: true }).click();
  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "Finance. Clearer. Closer.",
  );
  await expect(
    page.getByRole("textbox", { name: "Transaction date", exact: true }),
  ).toHaveValue("09/07/2026");
  await page.getByRole("button", { name: "Show toast", exact: true }).click();
  await expect(
    page.getByText("Copied · sample", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Close notification", exact: true })
    .click();
  await expect(
    page.getByText("Copied · sample", { exact: true }),
  ).not.toBeVisible();
});

test("locale switches reformat a valid date and retain an invalid draft", async ({
  page,
}) => {
  await page
    .getByRole("textbox", { name: "Дата операции", exact: true })
    .fill("29.02.2024");
  await page.getByRole("button", { name: "English", exact: true }).click();
  await expect(
    page.getByRole("textbox", { name: "Transaction date", exact: true }),
  ).toHaveValue("02/29/2024");
  await page
    .getByRole("textbox", { name: "Transaction date", exact: true })
    .fill("02/30/2024");
  await page.getByRole("button", { name: "Русский", exact: true }).click();
  await expect(
    page.getByRole("textbox", { name: "Дата операции", exact: true }),
  ).toHaveValue("02/30/2024");
  await page
    .getByRole("textbox", { name: "Дата операции", exact: true })
    .press("Tab");
  await expect(
    page.getByText("Проверьте дату. Формат: ДД.ММ.ГГГГ", { exact: true }),
  ).toBeVisible();
});

for (const locale of ["ru", "en"] as const) {
  test(`calendar clears an invalid date draft in ${locale}`, async ({
    page,
  }) => {
    const ru = locale === "ru";
    if (!ru)
      await page.getByRole("button", { name: "English", exact: true }).click();
    const date = page.getByRole("textbox", {
      name: ru ? "Дата операции" : "Transaction date",
      exact: true,
    });
    await date.fill(ru ? "31.02.2026" : "02/31/2026");
    await page
      .getByRole("button", {
        name: ru
          ? "Открыть календарь: Дата операции"
          : "Open calendar: Transaction date",
        exact: true,
      })
      .click();
    await expect(date).toHaveAttribute("aria-invalid", "true");
    await page
      .getByRole("button", { name: ru ? "Очистить" : "Clear", exact: true })
      .click();
    await expect(date).toHaveValue("");
    await expect(date).toHaveAttribute("aria-invalid", "false");
    await expect(page.locator('[data-slot="date-popover"]')).not.toBeVisible();
  });
}

test("rendered action, selection and field states meet text contrast", async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  const controls = [
    page.locator("#actions .wk-button-primary").first(),
    page.locator("#actions .wk-button-secondary").first(),
    page.locator("#actions .wk-button-destructive").first(),
    page.getByRole("button", { name: "Семья", exact: true }),
    page.getByLabel("Сумма", { exact: true }),
  ];
  await page.getByLabel("Сумма", { exact: true }).fill("9".repeat(257));
  for (const control of controls) {
    await page.getByRole("heading", { level: 1 }).click();
    for (const state of ["normal", "hover", "active", "focus"] as const) {
      if (state === "hover") await control.hover();
      if (state === "active") await page.mouse.down();
      if (state === "focus") {
        await page.mouse.up();
        await control.focus();
        await control.press("Tab");
        await page.keyboard.press("Shift+Tab");
      }
      const colors = await control.evaluate((element) => {
        const style = getComputedStyle(element);
        const canvas = document.createElement("canvas");
        canvas.width = canvas.height = 1;
        const context = canvas.getContext("2d")!;
        return [style.color, style.backgroundColor].map((color) => {
          context.clearRect(0, 0, 1, 1);
          context.fillStyle = color;
          context.fillRect(0, 0, 1, 1);
          return (
            "#" +
            [...context.getImageData(0, 0, 1, 1).data]
              .slice(0, 3)
              .map((channel) => channel.toString(16).padStart(2, "0"))
              .join("")
          );
        });
      });
      const hex = colors;
      expect(
        ThemeContract.contrast(hex[0], hex[1]),
        `${state}: ${colors}`,
      ).toBeGreaterThanOrEqual(4.5);
      await expect(control).toHaveCSS("border-width", "0px");
    }
  }
});

for (const [value, key] of [
  ["01.01.0001", "ArrowLeft"],
  ["31.12.9999", "ArrowRight"],
]) {
  test(`calendar preserves the supported boundary ${value}`, async ({
    page,
  }) => {
    const errors: string[] = [];
    page.on("pageerror", (error) => errors.push(error.message));
    const date = page.getByRole("textbox", {
      name: "Дата операции",
      exact: true,
    });
    await date.fill(value);
    await page
      .getByRole("button", { name: "Открыть календарь: Дата операции" })
      .click();
    await expect(
      page.locator('[data-slot="date-popover"] button:focus'),
    ).toHaveCount(1);
    await page.keyboard.press(key);
    await page.keyboard.press("Enter");
    await expect(date).toHaveValue(value);
    expect(errors).toEqual([]);
  });
}

test("portaled controls inherit the selected document language", async ({
  page,
}) => {
  for (const locale of ["ru", "en"]) {
    if (locale === "en")
      await page.getByRole("button", { name: "English", exact: true }).click();
    const ru = locale === "ru";
    await page
      .getByRole("button", {
        name: ru ? "Открыть форму" : "Open form",
        exact: true,
      })
      .click();
    const dialog = page.getByRole("dialog");
    expect(
      await dialog.evaluate((element) =>
        element.closest("[lang]")?.getAttribute("lang"),
      ),
    ).toBe(locale);
    await dialog
      .getByRole("combobox", { name: ru ? "Актив" : "Asset" })
      .click();
    const option = page.getByRole("option", { name: "BTC", exact: true });
    expect(
      await option.evaluate((element) =>
        element.closest("[lang]")?.getAttribute("lang"),
      ),
    ).toBe(locale);
    await option.click();
    await expect(page.getByRole("listbox")).not.toBeVisible();
    await page.keyboard.press("Escape");
    await expect(dialog).not.toBeVisible();
    await page
      .getByRole("button", {
        name: ru ? "Показать toast" : "Show toast",
        exact: true,
      })
      .click();
    const toast = page.locator(".wk-toast").last();
    await expect(toast).toBeVisible();
    expect(
      await toast.evaluate((element) =>
        element.closest("[lang]")?.getAttribute("lang"),
      ),
    ).toBe(locale);
    await toast
      .getByRole("button", {
        name: ru ? "Закрыть уведомление" : "Close notification",
      })
      .click();
  }
});

test("avatar names survive loaded images and failed-image fallback", async ({
  page,
}) => {
  const avatar = page.getByRole("img", {
    name: "Профиль с изображением",
    exact: true,
  });
  await expect(avatar).toHaveAccessibleName("Профиль с изображением");
  await expect(avatar.locator("img")).toBeVisible();
  await expect(
    page.getByRole("img", { name: "Участник А", exact: true }),
  ).toHaveCount(2);
  await page.route("**/brand/logo_512px.svg", (route) => route.abort());
  await page.reload();
  await expect(avatar).toHaveAccessibleName("Профиль с изображением");
  await expect(avatar.locator("img")).not.toBeVisible();
  await expect(avatar).toHaveText("WK");
});

test("unknown progress stays distinct from completion and is localized", async ({
  page,
}) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  const unknown = page.getByRole("progressbar", {
    name: "Обработка",
    exact: true,
  });
  const complete = page.getByRole("progressbar", {
    name: "Готово",
    exact: true,
  });
  await expect(unknown).toHaveAttribute(
    "aria-valuetext",
    "Объём пока неизвестен",
  );
  await expect(unknown).not.toHaveAttribute("aria-valuenow");
  await expect(unknown).toHaveAttribute("data-state", "indeterminate");
  await expect(unknown.locator('[data-slot="progress-indicator"]')).toHaveCSS(
    "animation-name",
    "none",
  );
  await expect(complete).toHaveAttribute("aria-valuenow", "100");
  await expect(complete).toHaveAttribute("data-state", "complete");
  await expect(
    complete.locator('[data-slot="progress-indicator"]'),
  ).toBeVisible();
  await page.getByRole("button", { name: "English", exact: true }).click();
  await expect(
    page.getByRole("progressbar", { name: "Processing", exact: true }),
  ).toHaveAttribute("aria-valuetext", "Total progress is not yet known");
});
