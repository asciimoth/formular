import assert from "node:assert/strict";
import test from "node:test";
import { JSDOM } from "jsdom";
import { FormularMenu } from "../src/formular-menu.js";

function setupDom() {
  const dom = new JSDOM("<!doctype html><html><head></head><body><div id=\"root\"></div></body></html>", {
    url: "https://example.test/"
  });
  globalThis.window = dom.window;
  globalThis.document = dom.window.document;
  globalThis.MutationObserver = dom.window.MutationObserver;
  Object.defineProperty(globalThis, "navigator", {
    configurable: true,
    value: dom.window.navigator
  });
  return dom;
}

function snapshot() {
  return {
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 7,
    blocks: [
      {
        id: "main",
        order: 1,
        generation: 3,
        form: false,
        items: [
          { type: "header", id: "h", text: "Settings" },
          { type: "field", id: "name", kind: "text", label: "Name", value: "Ada", validate: true },
          { type: "field", id: "enabled", kind: "checkbox", label: "Enabled", value: true },
          { type: "button", id: "save", label: "Save" }
        ]
      }
    ]
  };
}

test("renders a menu snapshot and sends field messages", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message));

  assert.equal(menu.feed(snapshot()), true);
  assert.match(document.body.textContent, /Settings/);
  outbox.length = 0;

  const input = document.querySelector("input[type='text']");
  input.value = "Grace";
  input.dispatchEvent(new window.Event("input", { bubbles: true }));

  assert.equal(outbox.length, 2);
  assert.equal(outbox[0].type, "field.validate");
  assert.equal(outbox[0].field.fieldId, "name");
  assert.equal(outbox[0].value, "Grace");
  assert.equal(outbox[1].type, "field.update");
  assert.equal(outbox[1].menuGeneration, 7);
  assert.equal(outbox[1].blockGeneration, 3);
});

test("form blocks apply collected values only when valid", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu(document.getElementById("root"), "settings", (message) => outbox.push(message));
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [{
      id: "form",
      order: 1,
      generation: 1,
      form: true,
      items: [
        { type: "field", id: "email", kind: "text", label: "Email", value: "a@example.com", required: true, validate: true, status: "ok" },
        { type: "field", id: "age", kind: "int", label: "Age", value: 41 }
      ]
    }]
  });
  outbox.length = 0;

  const apply = [...document.querySelectorAll("button")].find((button) => button.textContent === "Apply");
  assert.equal(apply.disabled, false);
  apply.click();

  assert.equal(outbox.length, 1);
  assert.deepEqual(outbox[0].values, { email: "a@example.com", age: 41 });
});

test("applies field status updates", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  menu.feed(snapshot());
  menu.feed({
    type: "field.status",
    menuId: "settings",
    field: { blockId: "main", fieldId: "name" },
    status: "error",
    statusText: "Bad value",
    readonly: true
  });

  assert.match(document.body.textContent, /Bad value/);
  assert.equal(document.querySelector("input[type='text']").disabled, true);
});

test("applies autocomplete hints to the focused datalist input", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message));
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [{
      id: "profile",
      order: 1,
      generation: 1,
      form: true,
      items: [
        { type: "field", id: "timezone", kind: "text", label: "Timezone", value: "UTC", autocomplete: { enabled: true, tag: "timezone" } }
      ]
    }]
  });

  const input = document.querySelector("input[list]");
  input.focus();
  input.value = "Europe/T";
  input.dispatchEvent(new window.Event("input", { bubbles: true }));
  menu.feed({
    type: "autocomplete.hints",
    menuId: "settings",
    menuGeneration: 1,
    blockGeneration: 1,
    field: { blockId: "profile", fieldId: "timezone" },
    prefix: "Europe/T",
    hints: ["Europe/T", "Europe/Tbilisi"]
  });

  const list = document.getElementById(input.getAttribute("list"));
  assert.deepEqual([...list.querySelectorAll("option")].map((option) => option.value), ["Europe/Tbilisi"]);
});

test("non-forced backend updates preserve local collapse state", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  const block = (collapsed, text = "Live") => ({
    id: "main",
    order: 1,
    generation: 1,
    form: false,
    collapsible: true,
    collapsed,
    items: [{ type: "label", id: "status", text }]
  });

  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [block(false)]
  });
  document.querySelector("button[title='Toggle block']").click();
  assert.equal(document.querySelector("button[title='Toggle block']").textContent, "+");
  assert.doesNotMatch(document.body.textContent, /Live/);

  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 2,
    blocks: [block(false, "Updated")]
  });
  assert.equal(document.querySelector("button[title='Toggle block']").textContent, "+");
  assert.doesNotMatch(document.body.textContent, /Updated/);

  menu.feed({
    type: "block.snapshot",
    menuId: "settings",
    menuGeneration: 2,
    blockGeneration: 2,
    block: block(false, "Patched")
  });
  assert.equal(document.querySelector("button[title='Toggle block']").textContent, "+");
  assert.doesNotMatch(document.body.textContent, /Patched/);

  document.querySelector("button[title='Toggle block']").click();
  assert.equal(document.querySelector("button[title='Toggle block']").textContent, "-");
  assert.match(document.body.textContent, /Patched/);
});

test("forced menu snapshots reset local collapse state", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  const block = {
    id: "main",
    order: 1,
    generation: 1,
    form: false,
    collapsible: true,
    collapsed: false,
    items: [{ type: "label", id: "status", text: "Live" }]
  };

  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [block]
  });
  document.querySelector("button[title='Toggle block']").click();

  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 2,
    force: true,
    blocks: [block]
  });
  assert.equal(document.querySelector("button[title='Toggle block']").textContent, "-");
  assert.match(document.body.textContent, /Live/);
});

test("block snapshots patch changed progress without interrupting active input", () => {
  setupDom();
  const outside = document.createElement("button");
  outside.textContent = "Outside";
  document.body.append(outside);
  const menu = new FormularMenu("root", "settings", () => {});
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    force: true,
    blocks: [{
      id: "main",
      order: 1,
      generation: 1,
      form: false,
      items: [
        { type: "progressbar", id: "sync", label: "Sync", progress: 10 },
        { type: "field", id: "name", kind: "text", label: "Name", value: "Ada" }
      ]
    }]
  });

  const input = document.querySelector("input[type='text']");
  input.focus();
  input.value = "Ada Lovelace";
  input.dispatchEvent(new window.Event("input", { bubbles: true }));
  outside.focus();

  menu.feed({
    type: "block.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blockGeneration: 1,
    block: {
      id: "main",
      order: 1,
      generation: 1,
      form: false,
      items: [
        { type: "progressbar", id: "sync", label: "Sync", progress: 20 },
        { type: "field", id: "name", kind: "text", label: "Name", value: "Ada" }
      ]
    }
  });

  assert.equal(document.activeElement, outside);
  assert.equal(input.value, "Ada Lovelace");
  assert.match(document.body.textContent, /20%/);
});

test("controls use current block state after in-place backend patches", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message));
  const block = (generation, label = "Run") => ({
    id: "main",
    order: 1,
    generation,
    form: false,
    items: [
      { type: "field", id: "name", kind: "text", label: "Name", value: "Ada", validate: true },
      { type: "button", id: "run", label }
    ]
  });

  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [block(1)]
  });
  outbox.length = 0;

  menu.feed({
    type: "block.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blockGeneration: 2,
    block: block(2, "Run now")
  });
  outbox.length = 0;
  document.querySelector("input[type='text']").value = "Grace";
  document.querySelector("input[type='text']").dispatchEvent(new window.Event("input", { bubbles: true }));
  [...document.querySelectorAll("button")].find((button) => button.textContent === "Run now").click();

  assert.equal(outbox[0].type, "field.validate");
  assert.equal(outbox[0].blockGeneration, 2);
  assert.equal(outbox[1].type, "field.update");
  assert.equal(outbox[1].blockGeneration, 2);
  assert.equal(outbox[2].type, "button.press");
  assert.equal(outbox[2].blockGeneration, 2);
});

test("form actions use current block state after in-place backend patches", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message));
  const block = (generation, age) => ({
    id: "form",
    order: 1,
    generation,
    form: true,
    items: [
      { type: "field", id: "email", kind: "text", label: "Email", value: "a@example.com", required: true, validate: true, status: "ok" },
      { type: "field", id: "age", kind: "int", label: "Age", value: age }
    ]
  });

  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [block(1, 41)]
  });
  outbox.length = 0;

  menu.feed({
    type: "block.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blockGeneration: 2,
    block: block(2, 42)
  });
  outbox.length = 0;
  [...document.querySelectorAll("button")].find((button) => button.textContent === "Apply").click();

  assert.equal(outbox.length, 1);
  assert.equal(outbox[0].type, "form.apply");
  assert.equal(outbox[0].blockGeneration, 2);
  assert.deepEqual(outbox[0].values, { email: "a@example.com", age: 42 });
});

test("form reset restores backend defaults after local edits", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "left", (message) => outbox.push(message));
  menu.feed({
    type: "menu.snapshot",
    menuId: "left",
    menuGeneration: 1,
    blocks: [{
      id: "log-submit",
      order: 1,
      generation: 1,
      form: true,
      items: [
        { type: "field", id: "level", kind: "radio", label: "Level", value: "info", allowedValues: ["trace", "debug", "info", "warn", "error", "panic"] },
        { type: "field", id: "message", kind: "text", label: "Message", value: "User submitted log line", required: true, validate: true }
      ]
    }]
  });
  outbox.length = 0;

  const message = document.querySelector("input[type='text']");
  message.value = "Changed message";
  message.dispatchEvent(new window.Event("input", { bubbles: true }));
  const error = [...document.querySelectorAll("input[type='radio']")].find((radio) => radio.value === JSON.stringify("error"));
  error.checked = true;
  error.dispatchEvent(new window.Event("change", { bubbles: true }));

  [...document.querySelectorAll("button")].find((button) => button.textContent === "Reset").click();

  assert.equal(document.querySelector("input[type='text']").value, "User submitted log line");
  const checked = [...document.querySelectorAll("input[type='radio']")].find((radio) => radio.checked);
  assert.equal(checked.value, JSON.stringify("info"));
  assert.equal(outbox.at(-1).type, "field.validate");
  assert.equal(outbox.at(-1).value, "User submitted log line");
});

test("copyable array fields copy current array values", async () => {
  setupDom();
  let copied = "";
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText: async (value) => { copied = value; } }
  });
  const menu = new FormularMenu("root", "settings", () => {});
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [{
      id: "live",
      order: 1,
      generation: 1,
      form: false,
      items: [{
        type: "field",
        id: "servers",
        kind: "array",
        label: "Servers",
        copyable: { text: "[server snapshot]" },
        templates: [
          { name: "http", items: [{ type: "field", id: "host", kind: "text", label: "Host", value: "new.local" }] },
          { name: "database", items: [{ type: "field", id: "dsn", kind: "text", label: "DSN", value: "postgres://localhost/app" }] }
        ],
        elements: [{
          id: "server-1",
          template: "http",
          items: [{ type: "field", id: "host", kind: "text", label: "Host", value: "localhost" }]
        }]
      }]
    }]
  });

  [...document.querySelectorAll(".formular-array-actions button")]
    .find((button) => button.textContent === "Copy")
    .click();
  await new Promise((resolve) => setTimeout(resolve, 0));

  assert.deepEqual(JSON.parse(copied), [{
    id: "server-1",
    template: "http",
    values: { host: "localhost" }
  }]);

  document.querySelector(".formular-array-actions select").value = "database";
  [...document.querySelectorAll(".formular-array-actions button")]
    .find((button) => button.textContent === "+")
    .click();
  [...document.querySelectorAll(".formular-array-actions button")]
    .find((button) => button.textContent === "Copy")
    .click();
  await new Promise((resolve) => setTimeout(resolve, 0));

  assert.deepEqual(JSON.parse(copied), [
    { id: "server-1", template: "http", values: { host: "localhost" } },
    { id: "local-1", template: "database", values: { dsn: "postgres://localhost/app" } }
  ]);
});

test("renders logs and patches appended log lines", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [{
      id: "main",
      order: 1,
      generation: 1,
      form: false,
      items: [
        { type: "logs", id: "events", label: "Events", logs: [{ level: "info", text: "Ready" }] }
      ]
    }]
  });

  assert.match(document.body.textContent, /\[info\]Ready/);
  assert.equal(document.querySelector("[data-level='info']").textContent, "[info]");

  menu.feed({
    type: "block.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blockGeneration: 1,
    block: {
      id: "main",
      order: 1,
      generation: 1,
      form: false,
      items: [
        {
          type: "logs",
          id: "events",
          label: "Events",
          logs: [
            { level: "info", text: "Ready" },
            { level: "error", text: "<script>alert(1)</script>" }
          ]
        }
      ]
    }
  });

  assert.match(document.body.textContent, /\[error\]<script>alert\(1\)<\/script>/);
  assert.equal(document.querySelectorAll("script").length, 0);
});

test("repeated logs snapshots do not corrupt sibling array fields", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  const block = (logs) => ({
    id: "main",
    order: 1,
    generation: 1,
    form: false,
    items: [
      { type: "logs", id: "events", label: "Events", logs },
      {
        type: "field",
        id: "rows",
        kind: "array",
        label: "Rows",
        templates: [{ name: "row", items: [{ type: "field", id: "name", kind: "text", label: "Name", value: "one" }] }],
        elements: [{ id: "row-1", template: "row", items: [{ type: "field", id: "name", kind: "text", label: "Name", value: "one" }] }]
      }
    ]
  });

  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [block([{ level: "info", text: "first" }])]
  });
  assert.doesNotThrow(() => {
    menu.feed({
      type: "block.snapshot",
      menuId: "settings",
      menuGeneration: 1,
      blockGeneration: 1,
      block: block([{ level: "info", text: "first" }, { level: "warn", text: "second" }])
    });
    menu.feed({
      type: "block.snapshot",
      menuId: "settings",
      menuGeneration: 1,
      blockGeneration: 1,
      block: block([{ level: "info", text: "first" }, { level: "warn", text: "second" }, { level: "error", text: "third" }])
    });
  });
  assert.match(document.body.textContent, /third/);
  assert.match(document.body.textContent, /Rows/);
});

test("ignores messages for other menus", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  assert.equal(menu.feed({ ...snapshot(), menuId: "other" }), false);
  assert.match(document.body.textContent, /No menu snapshot/);
});

test("renders backend text without executable HTML", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    blocks: [{
      id: "xss",
      order: 1,
      generation: 1,
      form: false,
      items: [
        { type: "header", id: "header", text: "<img src=x onerror=alert(1)>" },
        { type: "label", id: "plain", text: "<script>alert(1)</script>" },
        { type: "label", id: "code", format: "code", text: "<svg onload=alert(1)></svg>" },
        { type: "field", id: "field", kind: "text", label: "<b>Label</b>", value: "<img src=x onerror=alert(1)>", status: "error", statusText: "<script>alert(1)</script>" }
      ]
    }]
  });

  assert.equal(document.querySelectorAll("script,img,svg").length, 0);
  assert.match(document.body.textContent, /<img src=x onerror=alert\(1\)>/);
  assert.match(document.body.textContent, /<script>alert\(1\)<\/script>/);
});

test("markdown labels only create safe links", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    blocks: [{
      id: "xss",
      order: 1,
      generation: 1,
      form: false,
      items: [
        { type: "label", id: "bad-link", format: "markdown", text: "[click](javascript:alert(1)) **<img src=x>** `</code><script>alert(1)</script>`" },
        { type: "label", id: "good-link", format: "markdown", text: "[safe](https://example.com/path)" }
      ]
    }]
  });

  const links = [...document.querySelectorAll("a")];
  assert.equal(links[0].hasAttribute("href"), false);
  assert.equal(links[1].href, "https://example.com/path");
  assert.equal(document.querySelectorAll("script,img").length, 0);
});
