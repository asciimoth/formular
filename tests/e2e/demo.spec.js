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

test("demo evaluates item state conditions in the frontend", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const block = page.locator("#left-menu [data-block-id='frontend-state']");
  const advanced = block.getByLabel("Advanced note *");
  const action = block.getByRole("button", { name: "Conditional action" });
  const apply = block.getByRole("button", { name: "Apply" });
  const updates = page.locator("#message-log li", { hasText: "frontend -> middleware field.update" });
  const updatesBefore = await updates.count();

  await expect(advanced).toBeHidden();
  await expect(apply).toBeEnabled();
  await expect(action).toBeEnabled();

  await block.getByLabel("Show advanced").check();
  await expect(advanced).toBeVisible();
  await expect(apply).toBeDisabled();
  await advanced.fill("Local condition value");
  await expect(apply).toBeEnabled();

  await block.getByLabel("Action mode").fill("locked");
  await expect(action).toBeDisabled();
  await expect(updates).toHaveCount(updatesBefore);
});

test("demo dialogs complete backend message round trips", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();
  const left = page.locator("#left-menu");

  await left.getByRole("button", { name: "Yes/no dialog" }).click();
  let dialog = page.getByRole("dialog");
  await expect(dialog).toContainText("This yes/no dialog returns a boolean result.");
  await dialog.getByRole("button", { name: "Continue" }).click();
  await expect(left.getByText("Yes/no result: continue")).toBeVisible();

  await left.getByRole("button", { name: "Selection dialog" }).click();
  dialog = page.getByRole("dialog");
  await dialog.locator("select").selectOption(["go", "javascript"]);
  await dialog.getByRole("button", { name: "Use selection" }).click();
  await expect(left.getByText("Selection result: go, javascript")).toBeVisible();

  await left.getByRole("button", { name: "Captcha dialog" }).click();
  dialog = page.getByRole("dialog");
  await expect(dialog.locator("img")).toHaveAttribute("src", /^data:image\/svg\+xml;base64,/);
  await dialog.getByLabel("Captcha response").fill("4m7k");
  await dialog.getByRole("button", { name: "Submit" }).click();
  await expect(left.getByText("Captcha result: correct")).toBeVisible();

  await expect(page.getByText(/frontend -> middleware dialog\.response/).first()).toBeVisible();
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

test("left profile timezone field receives backend autocomplete hints", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const timezone = page.getByLabel("Timezone");
  await timezone.fill("Europe/T");

  await expect.poll(async () => timezone.evaluate((input) => {
    const list = document.getElementById(input.getAttribute("list"));
    return [...(list?.querySelectorAll("option") || [])].map((option) => option.value);
  })).toContain("Europe/Tbilisi");

  await timezone.fill("UTC");
  await expect.poll(async () => timezone.evaluate((input) => {
    const list = document.getElementById(input.getAttribute("list"));
    return [...(list?.querySelectorAll("option") || [])].map((option) => option.value);
  })).not.toContain("UTC");
});

test("left profile owner selector refreshes frontend-known users", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const owner = page.getByLabel("Owner");
  await expect(owner).toHaveValue(JSON.stringify("Ada"));
  await expect.poll(() => owner.locator("option").allTextContents()).toEqual(["Ada", "Grace", "Linus"]);

  await page.evaluate(() => window.formularDemoKnownUsers.push("Margaret"));
  await owner.dispatchEvent("pointerdown");
  await expect.poll(() => owner.locator("option").allTextContents()).toContain("Margaret");

  await owner.selectOption({ label: "Margaret" });
  await expect(owner).toHaveValue(JSON.stringify("Margaret"));
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

test("middleware cached snapshots survive repeated frontend reloads and further edits", async ({ page }) => {
  await page.goto("/demo/");
  const state = page.locator("#backend-state");
  await expect(state).toHaveText(/Go backend #\d+ running/);
  const backendState = await state.textContent();

  await page.getByLabel("Name *").fill("Cached Middleware User");
  await page.getByLabel("Email *").fill("cached@example.com");
  await expect(page.getByRole("button", { name: "Apply" }).first()).toBeEnabled();
  await page.getByRole("button", { name: "Apply" }).first().click();
  await expect(page.getByText("Form values accepted by Go WASM backend")).toBeVisible();

  await page.getByLabel("Enabled").uncheck();
  await page.locator("#right-menu").getByRole("radio", { name: "warn", exact: true }).check();
  await page.getByLabel("Volume").fill("67");

  await page.locator("#left-menu").getByRole("radio", { name: "error", exact: true }).check();
  await page.locator("#left-menu").getByLabel("Message").fill("Cached middleware log");
  await page.getByRole("button", { name: "Apply" }).nth(1).click();
  await expect(page.getByText("Cached middleware log")).toBeVisible();

  const array = page.locator("#right-menu .formular-array");
  await array.locator("select").selectOption("database");
  await array.getByRole("button", { name: "+" }).click();
  const database = page.locator("#right-menu .formular-element").filter({ hasText: "Servers: local-1" });
  await expect(database).toBeVisible();
  await database.getByRole("radio", { name: "mysql", exact: true }).check();
  await database.getByLabel("DSN").fill("mysql://cache.local/app");
  await database.getByLabel("Pool size").fill("31");

  let expectedVolume = "67";
  let expectedStatus = "warn";
  let queueAdded = false;
  const statusText = {
    ok: "Backend marked this field as OK",
    warn: "Backend marked this field as a warning",
    error: "Backend marked this field as an error"
  };

  for (const [index, nextStatus] of ["error", "ok", "warn"].entries()) {
    await page.getByRole("button", { name: "Restart frontends" }).click();
    await expect(state).toHaveText(backendState);

    await expect(page.getByLabel("Name *")).toHaveValue("Cached Middleware User");
    await expect(page.getByLabel("Email *")).toHaveValue("cached@example.com");
    await expect(page.getByLabel("Enabled")).not.toBeChecked();
    await expect(page.getByLabel("Volume")).toHaveValue(expectedVolume);
    await expect(page.locator("#right-menu").getByRole("radio", { name: expectedStatus, exact: true })).toBeChecked();
    await expect(page.getByText(statusText[expectedStatus])).toBeVisible();
    await expect(page.getByText("Cached middleware log")).toBeVisible();

    const cachedDatabase = page.locator("#right-menu .formular-element").filter({ hasText: "Servers: local-1" });
    await expect(cachedDatabase.getByRole("radio", { name: "mysql", exact: true })).toBeChecked();
    await expect(cachedDatabase.getByLabel("DSN")).toHaveValue("mysql://cache.local/app");
    await expect(cachedDatabase.getByLabel("Pool size")).toHaveValue("31");

    if (queueAdded) {
      const cachedQueue = page.locator("#right-menu .formular-element").filter({ hasText: "Servers: local-2" });
      await expect(cachedQueue.getByLabel("Subject")).toHaveValue("events.cached.reload");
    }

    expectedVolume = String(70 + index);
    expectedStatus = nextStatus;
    await page.getByLabel("Volume").fill(expectedVolume);
    await page.locator("#right-menu").getByRole("radio", { name: expectedStatus, exact: true }).check();
    await expect(page.getByText(statusText[expectedStatus])).toBeVisible();
    await page.getByRole("button", { name: "Refresh" }).click();
    await expect(page.getByText("Refresh button pressed")).toBeVisible();

    if (!queueAdded) {
      await page.locator("#right-menu .formular-array select").selectOption("queue");
      await page.locator("#right-menu .formular-array").getByRole("button", { name: "+" }).click();
      const queue = page.locator("#right-menu .formular-element").filter({ hasText: "Servers: local-2" });
      await expect(queue).toBeVisible();
      await queue.getByLabel("Subject").fill("events.cached.reload");
      queueAdded = true;
    }
  }

  await page.getByRole("button", { name: "Restart frontends" }).click();
  await expect(state).toHaveText(backendState);
  await expect(page.getByLabel("Volume")).toHaveValue(expectedVolume);
  await expect(page.locator("#right-menu").getByRole("radio", { name: expectedStatus, exact: true })).toBeChecked();
  await expect(page.getByText(statusText[expectedStatus])).toBeVisible();
  await expect(page.locator("#right-menu .formular-element").filter({ hasText: "Servers: local-2" }).getByLabel("Subject")).toHaveValue("events.cached.reload");
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

test("array element state conditions work for snapshots and new template elements", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const array = page.locator("#right-menu .formular-array");
  const server = array.locator(".formular-element").filter({ hasText: "Servers: server-1" });
  const serverCertificate = server.locator("[data-formular-item-id='certificate']");
  const serverTLS = server.locator("[data-formular-item-id='tls'] input[type='checkbox']");
  await expect(serverCertificate).toBeHidden();
  await expect(server.getByRole("button", { name: "Ping" })).toBeEnabled();

  await serverTLS.check();
  await expect(serverCertificate).toBeVisible();
  await server.getByLabel("Host").fill("locked");
  await expect(server.getByRole("button", { name: "Ping" })).toBeDisabled();

  await array.locator(".formular-array-actions select").selectOption("http");
  await array.getByRole("button", { name: "+" }).click();
  const local = array.locator(".formular-element").filter({ hasText: "Servers: local-1" });
  const localCertificate = local.locator("[data-formular-item-id='certificate']");
  const localTLS = local.locator("[data-formular-item-id='tls'] input[type='checkbox']");
  await expect(localCertificate).toBeHidden();
  await expect(local.getByRole("button", { name: "Ping" })).toBeEnabled();

  await localTLS.check();
  await expect(localCertificate).toBeVisible();
  await local.getByLabel("Host").fill("locked");
  await expect(local.getByRole("button", { name: "Ping" })).toBeDisabled();
});

test("array templates can add database element and survive frontend restart", async ({ page }) => {
  await page.goto("/demo/");
  await page.context().grantPermissions(["clipboard-read", "clipboard-write"], { origin: new URL(page.url()).origin });
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const copy = page.locator("#right-menu .formular-array .formular-array-actions").getByRole("button", { name: "Copy" });
  await expect(copy).toBeVisible();
  await copy.click();
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toContain("\"server-1\"");

  await page.locator(".formular-array select").selectOption("database");
  await page.getByRole("button", { name: "+" }).click();
  await expect(page.getByText(/Servers: local-/)).toBeVisible();
  await copy.click();
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toContain("\"local-1\"");

  await page.getByRole("radio", { name: "mysql", exact: true }).check();
  await page.getByLabel("DSN").fill("mysql://localhost/demo");
  await page.getByLabel("Pool size").fill("24");
  await copy.click();
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toContain("mysql://localhost/demo");
  await page.getByRole("button", { name: "Restart frontends" }).click();

  await expect(page.getByText(/Servers: local-/)).toBeVisible();
  await expect(page.getByRole("radio", { name: "mysql", exact: true })).toBeChecked();
  await expect(page.getByLabel("DSN")).toHaveValue("mysql://localhost/demo");
  await expect(page.getByLabel("Pool size")).toHaveValue("24");
});

test("array element generate button asks backend for fresh values", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const firstServer = page.locator("#right-menu .formular-element").filter({ hasText: "Servers: server-1" });
  await expect(firstServer.getByLabel("Host")).toHaveValue("localhost");
  await firstServer.getByRole("button", { name: "Generate" }).click();
  await expect(firstServer.getByLabel("Host")).toHaveValue("generated-1.local");
  await expect(firstServer.getByLabel("Port")).toHaveValue("8001");

  await page.locator(".formular-array select").selectOption("database");
  await page.getByRole("button", { name: "+" }).click();
  const generatedDatabase = page.locator("#right-menu .formular-element").filter({ hasText: /Servers: local-/ });
  await generatedDatabase.getByRole("button", { name: "Generate" }).click();

  await expect(generatedDatabase.getByLabel("DSN")).toHaveValue("postgres://generated-2.local/app");
  await expect(generatedDatabase.getByLabel("Pool size")).toHaveValue("7");
});

test("file input keeps selected file after frontend reads it", async ({ page }) => {
  await page.goto("/demo/");
  await expect(page.getByText(/Go backend #\d+ running/)).toBeVisible();

  const file = page.getByLabel("Avatar file");
  await file.setInputFiles(path.join(import.meta.dirname, "../fixtures/avatar.png"));

  await expect.poll(async () => file.evaluate((node) => node.files?.length || 0)).toBe(1);
  await expect.poll(async () => file.evaluate((node) => node.value.endsWith("avatar.png"))).toBe(true);
});
