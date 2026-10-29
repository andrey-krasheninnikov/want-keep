import { readFile } from "node:fs/promises";

import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { MemoryRouter } from "react-router";
import { AuthLayout } from "@/features/identity/AuthLayout";
import { LocaleProvider } from "@/locales/LocaleProvider";
import { LocaleController } from "@/locales/locale";

describe("application foundation", () => {
  it("renders the centered sign-in composition in the selected language", () => {
    const markup = renderToStaticMarkup(
      <MemoryRouter>
        <LocaleProvider controller={new LocaleController(undefined, "en")}>
          <AuthLayout login>
            <button>Log in with Passkeys</button>
          </AuthLayout>
        </LocaleProvider>
      </MemoryRouter>,
    );

    expect(markup.replace(/<[^>]+>/g, "")).toContain("Want Keep");
    expect(markup).toContain("Log in with Passkeys");
  });

  it("pins the shared design packages without the previous primitive stack", async () => {
    const manifest = JSON.parse(
      await readFile(new URL("../package.json", import.meta.url), "utf8"),
    );
    expect(manifest.dependencies["@want-foundation/desyst-foundation"]).toBe(
      "0.2.1",
    );
    expect(manifest.dependencies["@want-foundation/desyst-react"]).toBe(
      "0.2.1",
    );
    expect(manifest.dependencies["@base-ui/react"]).toBeUndefined();
    expect(manifest.dependencies["@daypicker/react"]).toBeUndefined();
    expect(manifest.devDependencies.shadcn).toBeUndefined();
  });
});
