import { expect, test } from "@playwright/test";

test("backend-provided content is rendered inertly", async ({ page }) => {
  await page.goto("/demo/");
  await page.evaluate(async () => {
    const { FormularMenu } = await import("/src/formular-menu.js");
    document.body.innerHTML = "<div id=\"root\"></div>";
    window.__formularXss = 0;
    const menu = new FormularMenu("root", "xss", () => {});
    menu.feed({
      type: "menu.snapshot",
      menuId: "xss",
      blocks: [{
        id: "payloads",
        order: 1,
        generation: 1,
        form: false,
        items: [
          { type: "header", id: "header", text: "<img src=x onerror=\"window.__formularXss=1\">" },
          { type: "label", id: "plain", text: "<script>window.__formularXss=2</script>" },
          { type: "label", id: "code", format: "code", text: "<svg onload=\"window.__formularXss=3\"></svg>" },
          { type: "label", id: "markdown", format: "markdown", text: "[bad](javascript:window.__formularXss=4) **<img src=x onerror=\"window.__formularXss=5\">** `</code><script>window.__formularXss=6</script>`" },
          { type: "field", id: "field", kind: "text", label: "<b>Unsafe label</b>", value: "<img src=x onerror=\"window.__formularXss=7\">" }
        ]
      }]
    });
    menu.feed({
      type: "field.status",
      menuId: "xss",
      field: { blockId: "payloads", fieldId: "field" },
      status: "error",
      statusText: "<img src=x onerror=\"window.__formularXss=8\">"
    });
  });

  await page.waitForTimeout(100);
  await expect.poll(() => page.evaluate(() => window.__formularXss)).toBe(0);
  await expect(page.locator("#root script,#root img,#root svg")).toHaveCount(0);
  await expect(page.locator("#root a").first()).not.toHaveAttribute("href", /javascript:/);
});
