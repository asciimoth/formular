import { expect, test } from "@playwright/test";

async function mountArrayHarness(page) {
  await page.goto("/demo/");
  await page.evaluate(async () => {
    document.body.innerHTML = "<main><div id=\"root\"></div></main>";
    const { FormularMenu } = await import("/src/formular-menu.js");
    const menu = new FormularMenu("root", "array-remove", () => {});
    window.arrayRemoveMenu = menu;
    window.arrayRemoveSnapshot = {
      type: "menu.snapshot",
      menuId: "array-remove",
      menuGeneration: 1,
      blocks: [{
        id: "settings",
        order: 1,
        generation: 1,
        form: true,
        items: [{
          type: "field",
          id: "servers",
          kind: "array",
          label: "Servers",
          elements: [
            {
              id: "server-1",
              template: "http",
              items: [{ type: "field", id: "host", kind: "text", label: "Host", value: "one.local" }]
            },
            {
              id: "server-2",
              template: "http",
              items: [{ type: "field", id: "host", kind: "text", label: "Host", value: "two.local" }]
            }
          ],
          templates: [{
            name: "http",
            label: "HTTP",
            items: [{ type: "field", id: "host", kind: "text", label: "Host", value: "new.local" }]
          }]
        }]
      }]
    };
    menu.feed(window.arrayRemoveSnapshot);
  });
}

async function serverTitles(page) {
  return page.locator(".formular-element-header strong").evaluateAll((nodes) => nodes.map((node) => node.textContent));
}

test("removing backend array element from form block survives dirty backend merge", async ({ page }) => {
  await mountArrayHarness(page);

  await page.getByRole("button", { name: "+" }).click();
  await expect(page.getByText("Servers: local-1")).toBeVisible();

  await page.locator(".formular-element").filter({ hasText: "Servers: server-1" }).getByTitle("Remove element").click();

  await expect(page.getByText("Servers: server-1")).toHaveCount(0);
  await expect.poll(() => serverTitles(page)).toEqual(["Servers: server-2", "Servers: local-1"]);

  await page.evaluate(() => {
    window.arrayRemoveMenu.feed({
      type: "block.snapshot",
      menuId: "array-remove",
      menuGeneration: 1,
      block: window.arrayRemoveSnapshot.blocks[0]
    });
  });

  await expect(page.getByText("Servers: server-1")).toHaveCount(0);
  await expect.poll(() => serverTitles(page)).toEqual(["Servers: server-2", "Servers: local-1"]);

  await page.getByRole("button", { name: "Reset" }).click();

  await expect.poll(() => serverTitles(page)).toEqual(["Servers: server-1", "Servers: server-2"]);
  await expect(page.getByText("Servers: local-1")).toHaveCount(0);
});
