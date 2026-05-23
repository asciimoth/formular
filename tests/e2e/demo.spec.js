import { expect, test } from "@playwright/test";
import path from "node:path";

test("demo boots Go WASM backend and renders both menus", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();
  await expect(page.getByRole("heading", { name: "Left frontend" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Right frontend" })).toBeVisible();
  await expect(page.getByText("Profile form")).toBeVisible();
  await expect(page.getByText("Realtime controls")).toBeVisible();

  await page.getByLabel("Email *").fill("invalid");
  await expect(page.getByText("Email must contain @")).toBeVisible();

  await page.getByRole("button", { name: "Refresh" }).click();
  await expect(page.getByText("Refresh button pressed")).toBeVisible();
  await expect(page.getByText("Activity log")).toBeVisible();
});

test("validated text input keeps focus while backend statuses arrive", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const email = page.getByLabel("Email *");
  await email.fill("");
  await email.pressSequentially("invalid@example.com", { delay: 30 });

  await expect(email).toBeFocused();
  await expect(page.getByText("Looks good").first()).toBeVisible();
  await expect(email).toHaveValue("invalid@example.com");
});

test("left progress updates do not interrupt profile input", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const name = page.getByLabel("Name *");
  await name.fill("");
  await name.pressSequentially("Typing during progress", { delay: 30 });
  await page.getByRole("heading", { name: "Left frontend" }).click();

  await expect(page.getByText("Background sync")).toBeVisible();
  await expect(page.getByText(/10%|20%|30%|40%|50%|60%|70%|80%|90%|100%/)).toBeVisible({ timeout: 3000 });
  await expect(name).toHaveValue("Typing during progress");
});

test("left form apply enables after initial validation and submits", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const apply = page.getByRole("button", { name: "Apply" }).first();
  await expect(apply).toBeEnabled();
  await apply.click();

  await expect(page.getByText("Form values accepted by Go WASM backend")).toBeVisible();
});

test("left log form appends submitted line to right logs", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  await page.locator("#left-menu").getByRole("radio", { name: "error", exact: true }).check();
  await page.locator("#left-menu").getByLabel("Message").fill("Submitted from left form");
  await page.getByRole("button", { name: "Apply" }).nth(1).click();

  await expect(page.getByText("Log line submitted")).toBeVisible();
  await expect(page.getByText("[error]")).toBeVisible();
  await expect(page.getByText("Submitted from left form")).toBeVisible();
});

test("frontend restart recreates menus without restarting backend", async ({ page }) => {
  await page.goto("/demo/");
  const state = page.locator("#backend-state");
  await expect(state).toHaveText(/Go backend #\d+ running/);
  const before = await state.textContent();

  await page.getByLabel("Name *").fill("Changed before frontend restart");
  await page.getByLabel("Email *").fill("changed@example.com");
  await expect(page.getByRole("button", { name: "Apply" }).first()).toBeEnabled();
  await page.getByRole("button", { name: "Apply" }).first().click();
  await expect(page.getByText("Form values accepted by Go WASM backend")).toBeVisible();

  await page.getByLabel("Enabled").uncheck();
  await page.getByLabel("fast").check();
  await page.getByLabel("Volume").fill("73");
  await page.getByRole("button", { name: "Restart frontends" }).click();

  await expect(state).toHaveText(before);
  await expect(page.getByLabel("Name *")).toHaveValue("Changed before frontend restart");
  await expect(page.getByLabel("Email *")).toHaveValue("changed@example.com");
  await expect(page.getByLabel("Enabled")).not.toBeChecked();
  await expect(page.getByLabel("fast")).toBeChecked();
  await expect(page.getByLabel("Volume")).toHaveValue("73");
  await expect(page.getByRole("button", { name: "Apply" }).first()).toBeEnabled();
});

test("backend restart resends snapshots without recreating frontends", async ({ page }) => {
  await page.goto("/demo/");
  const state = page.locator("#backend-state");
  await expect(state).toHaveText(/Go backend #1 running/);

  await page.getByRole("button", { name: "Restart backend" }).click();

  await expect(state).toHaveText("Go backend #2 running");
  await expect(page.getByText("Profile form")).toBeVisible();
  await expect(page.getByText("Realtime controls")).toBeVisible();
  await expect(page.getByRole("button", { name: "Apply" }).first()).toBeEnabled();
});

test("right array input keeps focus while backend block snapshots arrive", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const host = page.getByLabel("Host");
  await host.fill("");
  await host.pressSequentially("backend-owned.local", { delay: 30 });

  await expect(host).toBeFocused();
  await expect(host).toHaveValue("backend-owned.local");
  await expect(page.getByText("Realtime update received")).toBeVisible();
});

test("backend validation demo updates field status from radio changes", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  await page.locator("#right-menu").getByRole("radio", { name: "warn", exact: true }).check();
  await expect(page.getByText("Backend marked this field as a warning")).toBeVisible();

  await page.locator("#right-menu").getByRole("radio", { name: "error", exact: true }).check();
  await expect(page.getByText("Backend marked this field as an error")).toBeVisible();
  await expect(page.getByLabel("Backend validated input")).toHaveValue("Change the radio below");
});

test("array templates can add database element and survive frontend restart", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  await page.locator(".formular-array select").selectOption("database");
  await page.getByRole("button", { name: "+" }).click();
  await expect(page.getByText(/Servers: local-/)).toBeVisible();

  await page.getByRole("radio", { name: "mysql", exact: true }).check();
  await page.getByLabel("DSN").fill("mysql://localhost/demo");
  await page.getByLabel("Pool size").fill("24");
  await page.getByRole("button", { name: "Restart frontends" }).click();

  await expect(page.getByText(/Servers: local-/)).toBeVisible();
  await expect(page.getByRole("radio", { name: "mysql", exact: true })).toBeChecked();
  await expect(page.getByLabel("DSN")).toHaveValue("mysql://localhost/demo");
  await expect(page.getByLabel("Pool size")).toHaveValue("24");
});

test("file input keeps selected file after frontend reads it", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const file = page.getByLabel("Avatar file");
  await file.setInputFiles(path.join(import.meta.dirname, "../fixtures/avatar.png"));

  await expect.poll(async () => file.evaluate((node) => node.files?.length || 0)).toBe(1);
  await expect.poll(async () => file.evaluate((node) => node.value.endsWith("avatar.png"))).toBe(true);
});
