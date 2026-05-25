import { expect, test } from "@playwright/test";
import path from "node:path";

async function mountResetHarness(page) {
  await page.goto("/demo/");
  await page.evaluate(async () => {
    document.body.innerHTML = "<main><div id=\"root\"></div></main>";
    const { FormularMenu } = await import("/src/formular-menu.js");
    const menu = new FormularMenu("root", "reset", () => {});
    menu.feed({
      type: "menu.snapshot",
      menuId: "reset",
      menuGeneration: 1,
      blocks: [{
        id: "controls",
        order: 1,
        generation: 1,
        form: true,
        items: [
          { type: "field", id: "text", kind: "text", label: "Text", value: "default text" },
          { type: "field", id: "secret", kind: "text", label: "Secret", value: "default secret", secret: true },
          { type: "field", id: "notes", kind: "text", label: "Notes", value: "line one\nline two", multiline: true },
          { type: "field", id: "count", kind: "int", label: "Count", value: 7 },
          { type: "field", id: "ratio", kind: "float", label: "Ratio", value: 1.5 },
          { type: "field", id: "volume", kind: "range", label: "Volume", value: 42, min: 0, max: 100 },
          { type: "field", id: "enabled", kind: "checkbox", label: "Enabled", value: true },
          { type: "field", id: "level", kind: "radio", label: "Level", value: "info", allowedValues: ["debug", "info", "error"] },
          { type: "field", id: "mode", kind: "text", label: "Mode", value: "balanced", allowedValues: ["fast", "balanced", "safe"] },
          { type: "field", id: "attachment", kind: "file", label: "Attachment", value: null },
          {
            type: "field",
            id: "servers",
            kind: "array",
            label: "Servers",
            elements: [{
              id: "server-1",
              template: "http",
              items: [
                { type: "field", id: "host", kind: "text", label: "Host", value: "localhost" },
                { type: "field", id: "tls", kind: "checkbox", label: "TLS", value: false }
              ]
            }],
            templates: [{
              name: "http",
              label: "HTTP",
              items: [
                { type: "field", id: "host", kind: "text", label: "Host", value: "new.local" },
                { type: "field", id: "tls", kind: "checkbox", label: "TLS", value: true }
              ]
            }]
          }
        ]
      }]
    });
  });
}

test("form reset restores backend defaults for every control type", async ({ page }) => {
  await mountResetHarness(page);

  await page.getByLabel("Text").fill("changed text");
  await page.getByLabel("Secret").fill("changed secret");
  await page.getByLabel("Notes").fill("changed\nnotes");
  await page.getByLabel("Count").fill("99");
  await page.getByLabel("Ratio").fill("3.25");
  await page.getByLabel("Volume").evaluate((node) => {
    node.value = "73";
    node.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await page.getByLabel("Enabled").uncheck();
  await page.getByRole("radio", { name: "error", exact: true }).check();
  await page.getByLabel("Mode").selectOption(JSON.stringify("fast"));
  const attachment = page.getByLabel("Attachment");
  await attachment.setInputFiles(path.join(import.meta.dirname, "../fixtures/avatar.png"));
  await page.getByLabel("Host").fill("changed.local");
  await page.getByLabel("TLS").check();
  await page.getByRole("button", { name: "+" }).click();
  await expect(page.getByText("Servers: local-1")).toBeVisible();

  await page.getByRole("button", { name: "Reset" }).click();

  await expect(page.getByLabel("Text")).toHaveValue("default text");
  await expect(page.getByLabel("Secret")).toHaveValue("default secret");
  await expect(page.getByLabel("Notes")).toHaveValue("line one\nline two");
  await expect(page.getByLabel("Count")).toHaveValue("7");
  await expect(page.getByLabel("Ratio")).toHaveValue("1.5");
  await expect(page.getByLabel("Volume")).toHaveValue("42");
  await expect(page.getByLabel("Enabled")).toBeChecked();
  await expect(page.getByRole("radio", { name: "info", exact: true })).toBeChecked();
  await expect(page.getByLabel("Mode")).toHaveValue(JSON.stringify("balanced"));
  await expect.poll(async () => attachment.evaluate((node) => node.files?.length || 0)).toBe(0);
  await expect(page.getByLabel("Host")).toHaveValue("localhost");
  await expect(page.getByLabel("TLS")).not.toBeChecked();
  await expect(page.getByText("Servers: server-1")).toBeVisible();
  await expect(page.getByText("Servers: local-1")).toHaveCount(0);
});
