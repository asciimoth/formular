import { expect, test } from "@playwright/test";

test("opens a native modal captcha dialog and sends its response", async ({ page }) => {
  await page.goto("/demo/");
  await page.evaluate(async () => {
    const { FormularMenu } = await import("/src/formular-menu.js");
    const root = document.createElement("div");
    root.id = "dialog-test-root";
    document.body.append(root);
    window.dialogTestOutbox = [];
    window.dialogTestMenu = new FormularMenu(root, "dialog-test", (message) => {
      window.dialogTestOutbox.push(message);
    });
    window.dialogTestMenu.feed({
      type: "dialog.create",
      menuId: "dialog-test",
      menuGeneration: 9,
      dialog: {
        id: "captcha",
        kind: "captcha",
        title: "Verify",
        text: "Enter the image text.",
        resources: [{
          id: "challenge",
          mimeType: "image/png",
          data: "cG5n",
          alt: "Captcha challenge"
        }]
      }
    });
  });

  const dialog = page.locator("#dialog-test-root dialog");
  await expect(dialog).toBeVisible();
  await expect.poll(() => dialog.evaluate((node) => node.matches(":modal"))).toBe(true);
  await expect(dialog.locator("img")).toHaveAttribute("src", "data:image/png;base64,cG5n");

  await dialog.getByLabel("Captcha response").fill("A7Bc");
  await dialog.getByRole("button", { name: "Submit" }).click();

  await expect(dialog).toHaveCount(0);
  await expect.poll(() => page.evaluate(() => window.dialogTestOutbox)).toEqual([{
    type: "dialog.response",
    menuId: "dialog-test",
    menuGeneration: 9,
    dialogId: "captcha",
    value: "A7Bc"
  }]);
});
