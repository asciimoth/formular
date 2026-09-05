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
  Object.defineProperties(dom.window.HTMLDialogElement.prototype, {
    showModal: {
      configurable: true,
      value() {
        this.setAttribute("open", "");
        this.__formularShowModalCalled = true;
      }
    },
    close: {
      configurable: true,
      value() {
        this.removeAttribute("open");
        this.dispatchEvent(new dom.window.Event("close"));
      }
    }
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

test("updates conditional visibility and readonly state without backend messages", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message));
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
        { type: "field", id: "advanced", kind: "checkbox", label: "Advanced", value: false },
        { type: "field", id: "lock", kind: "text", label: "Lock", value: "edit" },
        {
          type: "field",
          id: "details",
          kind: "text",
          label: "Details",
          value: "",
          required: true,
          stateConditions: {
            visible: {
              all: [
                { source: { fieldId: "advanced" }, operator: "equals", value: true },
                { source: { fieldId: "lock" }, operator: "notEquals", value: "blocked" }
              ]
            }
          }
        },
        {
          type: "button",
          id: "save",
          label: "Save draft",
          stateConditions: {
            readonly: { source: { fieldId: "lock" }, operator: "equals", value: "locked" }
          }
        },
        {
          type: "label",
          id: "summary",
          text: "Conditional summary",
          stateConditions: {
            visible: {
              any: [
                { source: { fieldId: "advanced" }, operator: "equals", value: true },
                { not: { source: { fieldId: "lock" }, operator: "notEmpty" } }
              ]
            }
          }
        },
        {
          type: "label",
          id: "empty-lock",
          text: "Lock is empty",
          stateConditions: {
            visible: { source: { fieldId: "lock" }, operator: "empty" }
          }
        },
        {
          type: "label",
          id: "missing-source",
          text: "Missing source",
          stateConditions: {
            visible: { source: { fieldId: "unknown" }, operator: "equals", value: true }
          }
        },
        {
          type: "button",
          id: "inactive",
          label: "Always inactive",
          inactive: true,
          stateConditions: {
            readonly: { source: { fieldId: "advanced" }, operator: "equals", value: true }
          }
        }
      ]
    }]
  });

  const item = (id) => document.querySelector(`[data-formular-item-id='${id}']`);
  const apply = [...document.querySelectorAll("button")].find((button) => button.textContent === "Apply");
  const save = item("save");
  const advanced = item("advanced").querySelector("input");
  const lock = item("lock").querySelector("input");

  assert.equal(item("details").hidden, true);
  assert.equal(item("summary").hidden, true);
  assert.equal(item("empty-lock").hidden, true);
  assert.equal(item("missing-source").hidden, true);
  assert.equal(apply.disabled, false, "a hidden required field must not block apply");
  assert.equal(save.disabled, false);
  assert.equal(item("inactive").disabled, true, "static inactive state must have priority");

  advanced.checked = true;
  advanced.dispatchEvent(new window.Event("change", { bubbles: true }));
  assert.equal(item("details").hidden, false);
  assert.equal(item("summary").hidden, false);
  assert.equal(apply.disabled, true, "the visible required field must block apply");

  lock.focus();
  lock.value = "";
  lock.dispatchEvent(new window.Event("input", { bubbles: true }));
  assert.equal(item("empty-lock").hidden, false);
  lock.value = "locked";
  lock.dispatchEvent(new window.Event("input", { bubbles: true }));
  assert.equal(document.activeElement, lock, "state evaluation must not rebuild the source control");
  assert.equal(item("empty-lock").hidden, true);
  assert.equal(save.disabled, true);
  assert.equal(outbox.length, 0, "form-local state changes must not need a backend round trip");
});

test("resolves state sources across blocks and inside array elements", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [
      {
        id: "controls",
        order: 1,
        generation: 1,
        form: true,
        items: [
          { type: "field", id: "showStatus", kind: "checkbox", label: "Show status", value: false }
        ]
      },
      {
        id: "content",
        order: 2,
        generation: 1,
        form: true,
        items: [
          {
            type: "label",
            id: "status",
            text: "Cross-block status",
            stateConditions: {
              visible: {
                source: { blockId: "controls", fieldId: "showStatus" },
                operator: "equals",
                value: true
              }
            }
          },
          {
            type: "field",
            id: "servers",
            kind: "array",
            label: "Servers",
            templates: [],
            elements: [{
              id: "server-1",
              template: "http",
              items: [
                { type: "field", id: "tls", kind: "checkbox", label: "TLS", value: false },
                {
                  type: "field",
                  id: "certificate",
                  kind: "text",
                  label: "Certificate",
                  value: "",
                  stateConditions: {
                    visible: { source: { fieldId: "tls" }, operator: "equals", value: true }
                  }
                }
              ]
            }]
          }
        ]
      }
    ]
  });

  const status = document.querySelector("[data-formular-item-id='status']");
  const certificate = document.querySelector("[data-formular-item-id='certificate']");
  const add = document.querySelector("[data-formular-item-id='servers'] button[title='Add element']");
  assert.equal(status.hidden, true);
  assert.equal(certificate.hidden, true);
  assert.equal(add.disabled, true, "state refresh must preserve intrinsic control state");

  const showStatus = document.querySelector("[data-block-id='controls'] input[type='checkbox']");
  showStatus.checked = true;
  showStatus.dispatchEvent(new window.Event("change", { bubbles: true }));
  assert.equal(status.hidden, false);

  const tls = document.querySelector("[data-formular-item-id='tls'] input[type='checkbox']");
  tls.checked = true;
  tls.dispatchEvent(new window.Event("change", { bubbles: true }));
  assert.equal(certificate.hidden, false);
});

test("multiline text fields do not wrap and expand to fit their content", async () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  const data = snapshot();
  data.blocks[0].items.push({
    type: "field",
    id: "notes",
    kind: "text",
    label: "Notes",
    value: "line one\nline two",
    multiline: true
  });

  menu.feed(data);
  await Promise.resolve();

  const textarea = document.querySelector("textarea");
  assert.equal(textarea.wrap, "off");
  assert.equal(textarea.style.overflowY, "hidden");

  Object.defineProperties(textarea, {
    offsetHeight: { configurable: true, value: 90 },
    clientHeight: { configurable: true, value: 88 },
    scrollHeight: { configurable: true, value: 140 }
  });
  textarea.value += "\nline three";
  textarea.dispatchEvent(new window.Event("input", { bubbles: true }));

  assert.equal(textarea.style.height, "142px");
});

test("renders visible help markers for item help", () => {
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
        { type: "header", id: "title", text: "Settings", help: "Section help" },
        { type: "label", id: "intro", text: "Intro", help: "Label help" },
        { type: "progressbar", id: "sync", label: "Sync", progress: 10, help: "Progress help" },
        { type: "logs", id: "events", label: "Events", logs: [], help: "Logs help" },
        { type: "button", id: "run", label: "Run", help: "Button help" },
        { type: "field", id: "name", kind: "text", label: "Name", value: "Ada", help: "Field help" }
      ]
    }]
  });

  const markers = [...document.querySelectorAll(".formular-help-marker")];
  assert.equal(markers.length, 6);
  assert.deepEqual(markers.map((marker) => marker.title), [
    "Section help",
    "Label help",
    "Progress help",
    "Logs help",
    "Button help",
    "Field help"
  ]);
  assert.doesNotMatch(document.body.textContent, /Field help/);
  assert.equal(document.querySelector(".formular-help"), null);
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
  assert.equal(document.querySelector("input[type='text']").dataset.status, "ok");
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
  assert.equal(document.querySelector("input[type='text']").dataset.status, "error");
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

test("renders frontend-provided selector values for a marked text field", () => {
  setupDom();
  const outbox = [];
  let users = ["Ada", "Grace"];
  let selectorCalls = 0;
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message), {
    selectors: (selector) => {
      selectorCalls += 1;
      return selector === "users" ? users : undefined;
    }
  });
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blocks: [{
      id: "profile",
      order: 1,
      generation: 1,
      form: false,
      items: [
        { type: "field", id: "owner", kind: "text", label: "Owner", value: "Ada", selector: "users" },
        { type: "field", id: "assignee", kind: "text", label: "Assignee", value: "Linus", selector: "users" },
        { type: "field", id: "reviewer", kind: "text", label: "Reviewer", value: "", selector: "unknown" }
      ]
    }]
  });

  const select = document.querySelector("select");
  assert.deepEqual([...select.options].map((option) => option.textContent), ["Ada", "Grace"]);
  assert.equal(select.value, JSON.stringify("Ada"));
  const assignee = document.querySelectorAll("select")[1];
  assert.deepEqual([...assignee.options].map((option) => option.textContent), ["Linus", "Ada", "Grace"]);
  assert.equal(assignee.value, JSON.stringify("Linus"));
  assert.equal(document.querySelectorAll("input[type='text']").length, 1);
  assert.equal(selectorCalls, 3);

  select.value = JSON.stringify("Grace");
  select.dispatchEvent(new window.Event("change", { bubbles: true }));
  assert.equal(outbox.at(-1).type, "field.update");
  assert.equal(outbox.at(-1).value, "Grace");

  users = ["Grace", "Linus"];
  select.dispatchEvent(new window.PointerEvent("pointerdown", { bubbles: true }));
  select.click();
  assert.equal(selectorCalls, 4);
  assert.deepEqual([...select.options].map((option) => option.textContent), ["Grace", "Linus"]);
  assert.equal(select.value, JSON.stringify("Grace"));

  users = ["Grace", "Margaret"];
  select.click();
  assert.equal(selectorCalls, 5);
  assert.deepEqual([...select.options].map((option) => option.textContent), ["Grace", "Margaret"]);

  menu.feed({
    type: "block.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    blockGeneration: 2,
    block: {
      id: "profile",
      order: 1,
      generation: 2,
      form: false,
      items: [
        { type: "field", id: "owner", kind: "text", label: "Owner", value: "Ada" },
        { type: "field", id: "assignee", kind: "text", label: "Assignee", value: "Linus", selector: "users" },
        { type: "field", id: "reviewer", kind: "text", label: "Reviewer", value: "", selector: "unknown" }
      ]
    }
  });
  assert.equal(document.querySelectorAll("select").length, 1);
  assert.equal(document.querySelectorAll("input[type='text']").length, 2);
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

test("array fields hide the selector when only one template is available", () => {
  setupDom();
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
        templates: [{
          name: "http",
          items: [{ type: "field", id: "host", kind: "text", label: "Host", value: "new.local" }]
        }]
      }]
    }]
  });

  assert.equal(document.querySelector(".formular-array-actions select"), null);
  document.querySelector(".formular-array-actions button").click();
  assert.equal(document.querySelector(".formular-element input").value, "new.local");
});

test("regular block snapshots update clean array field values", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  const block = (host) => ({
    id: "live",
    order: 1,
    generation: 1,
    form: false,
    items: [{
      type: "field",
      id: "servers",
      kind: "array",
      label: "Servers",
      elements: [{
        id: "server-1",
        template: "http",
        items: [{ type: "field", id: "host", kind: "text", label: "Host", value: host }]
      }]
    }]
  });

  menu.feed({ type: "menu.snapshot", menuId: "settings", menuGeneration: 1, blocks: [block("localhost")] });
  menu.feed({ type: "block.snapshot", menuId: "settings", menuGeneration: 1, blockGeneration: 2, block: block("generated.local") });

  assert.equal(document.querySelector("input[type='text']").value, "generated.local");
});

test("regular block snapshots preserve dirty nested array field values", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  const block = (host) => ({
    id: "live",
    order: 1,
    generation: 1,
    form: false,
    items: [{
      type: "field",
      id: "servers",
      kind: "array",
      label: "Servers",
      elements: [{
        id: "server-1",
        template: "http",
        items: [{ type: "field", id: "host", kind: "text", label: "Host", value: host }]
      }]
    }]
  });

  menu.feed({ type: "menu.snapshot", menuId: "settings", menuGeneration: 1, blocks: [block("localhost")] });
  const host = document.querySelector("input[type='text']");
  host.value = "local-edit.local";
  host.dispatchEvent(new window.Event("input", { bubbles: true }));
  menu.feed({ type: "block.snapshot", menuId: "settings", menuGeneration: 1, blockGeneration: 2, block: block("backend.local") });

  assert.equal(document.querySelector("input[type='text']").value, "local-edit.local");
});

test("regular block snapshots update clean fields inside locally added array elements", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  const block = (dsn) => ({
    id: "live",
    order: 1,
    generation: 1,
    form: false,
    items: [{
      type: "field",
      id: "servers",
      kind: "array",
      label: "Servers",
      templates: [{ name: "database", items: [{ type: "field", id: "dsn", kind: "text", label: "DSN", value: "postgres://localhost/app" }] }],
      elements: dsn ? [{
        id: "local-1",
        template: "database",
        items: [{ type: "field", id: "dsn", kind: "text", label: "DSN", value: dsn }]
      }] : []
    }]
  });

  menu.feed({ type: "menu.snapshot", menuId: "settings", menuGeneration: 1, blocks: [block("")] });
  document.querySelector(".formular-array-actions button").click();
  menu.feed({ type: "block.snapshot", menuId: "settings", menuGeneration: 1, blockGeneration: 2, block: block("postgres://generated.local/app") });

  assert.equal(document.querySelector("input[type='text']").value, "postgres://generated.local/app");
});

test("cached local array elements advance new local ids after force snapshots", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message));
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    menuGeneration: 1,
    force: true,
    blocks: [{
      id: "main",
      order: 1,
      generation: 1,
      items: [{
        type: "field",
        id: "items",
        kind: "array",
        label: "Items",
        templates: [{ name: "entry", items: [{ type: "field", id: "name", kind: "text", label: "Name", value: "" }] }],
        elements: [{ id: "local-3", template: "entry", items: [{ type: "field", id: "name", kind: "text", label: "Name", value: "cached" }] }]
      }]
    }]
  });

  document.querySelector(".formular-icon").click();

  assert.match(document.body.textContent, /Items: local-4/);
  assert.equal(outbox.at(-1).value.at(-1).id, "local-4");
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

test("markdown labels render simple inline formatting", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    blocks: [{
      id: "docs",
      order: 1,
      generation: 1,
      form: false,
      items: [
        { type: "label", id: "intro", format: "markdown", text: "Read **carefully**, run `formular`, then visit [docs](/docs)." }
      ]
    }]
  });

  const label = document.querySelector(".formular-label");
  assert.equal(label.querySelector("strong").textContent, "carefully");
  assert.equal(label.querySelector("code").textContent, "formular");
  assert.equal(label.querySelector("a").textContent, "docs");
  assert.equal(label.querySelector("a").href, "https://example.test/docs");
  assert.equal(label.textContent, "Read carefully, run formular, then visit docs.");
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

test("backend-controlled labels, options, and statuses remain inert", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});
  menu.feed({
    type: "menu.snapshot",
    menuId: "settings",
    blocks: [{
      id: "<img src=x onerror=alert(1)>",
      order: 1,
      generation: 1,
      form: false,
      items: [
        { type: "progressbar", id: "progress", label: "<svg onload=alert(2)>", progress: 25, help: "<img src=x onerror=alert(3)>" },
        { type: "logs", id: "logs", label: "<script>alert(4)</script>", logs: [{ level: "\"><img src=x onerror=alert(5)>", text: "<iframe src=javascript:alert(6)></iframe>" }] },
        { type: "button", id: "button", label: "<img src=x onerror=alert(7)>", help: "<script>alert(8)</script>" },
        { type: "field", id: "choice", kind: "text", label: "<b>Choice</b>", value: "<option>", allowedValues: ["<img src=x onerror=alert(9)>"] },
        {
          type: "field",
          id: "items",
          kind: "array",
          label: "<script>alert(10)</script>",
          elements: [{ id: "<img src=x onerror=alert(11)>", template: "row", items: [] }],
          templates: [{ name: "<svg onload=alert(12)>", label: "<img src=x onerror=alert(13)>", items: [] }]
        }
      ]
    }]
  });
  menu.feed({
    type: "field.status",
    menuId: "settings",
    field: { blockId: "<img src=x onerror=alert(1)>", fieldId: "choice" },
    status: "\"><img src=x onerror=alert(14)>",
    statusText: "<script>alert(15)</script>"
  });

  assert.equal(document.querySelectorAll("script,img,svg,iframe").length, 0);
  assert.match(document.body.textContent, /<svg onload=alert\(2\)>/);
  assert.match(document.body.textContent, /<iframe src=javascript:alert\(6\)><\/iframe>/);
  assert.match(document.body.textContent, /<script>alert\(15\)<\/script>/);
});

test("yes/no dialogs use showModal and send one boolean response", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message));

  assert.equal(menu.feed({
    type: "dialog.create",
    menuId: "settings",
    menuGeneration: 7,
    dialog: {
      id: "delete",
      kind: "yesno",
      title: "Delete item?",
      text: "This cannot be undone.",
      yesLabel: "Delete",
      noLabel: "Keep"
    }
  }), true);

  const dialog = document.querySelector("dialog");
  assert.ok(dialog);
  assert.equal(dialog.open, true);
  assert.equal(dialog.__formularShowModalCalled, true);
  assert.match(dialog.textContent, /This cannot be undone/);

  const yes = [...dialog.querySelectorAll("button")].find((button) => button.textContent === "Delete");
  yes.click();

  assert.deepEqual(outbox, [{
    type: "dialog.response",
    menuId: "settings",
    menuGeneration: 7,
    dialogId: "delete",
    value: true
  }]);
  assert.equal(document.querySelector("dialog"), null);
});

test("selection dialogs support multiple selected values", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message));

  menu.feed({
    type: "dialog.create",
    menuId: "settings",
    menuGeneration: 3,
    dialog: {
      id: "regions",
      kind: "selection",
      title: "Regions",
      options: [
        { value: "eu", label: "Europe", selected: true },
        { value: "na", label: "North America" },
        { value: "apac", label: "Asia Pacific", selected: true }
      ],
      multiple: true,
      submitLabel: "Use regions"
    }
  });

  const dialog = document.querySelector("dialog");
  const select = dialog.querySelector("select");
  assert.equal(select.multiple, true);
  assert.deepEqual([...select.selectedOptions].map((option) => option.value), ["eu", "apac"]);
  [...dialog.querySelectorAll("button")].find((button) => button.textContent === "Use regions").click();

  assert.deepEqual(outbox.at(-1).value, ["eu", "apac"]);
});

test("captcha dialogs render attached base64 images and return text input", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message));

  menu.feed({
    type: "dialog.create",
    menuId: "settings",
    menuGeneration: 5,
    dialog: {
      id: "captcha-1",
      kind: "captcha",
      title: "<img src=x onerror=alert(1)>",
      text: "Enter the text in the image.",
      placeholder: "Captcha text",
      resources: [{
        id: "challenge",
        mimeType: "image/png",
        data: "cG5n",
        alt: "Captcha challenge"
      }]
    }
  });

  const dialog = document.querySelector("dialog");
  const image = dialog.querySelector("img");
  const input = dialog.querySelector("input");
  const submit = [...dialog.querySelectorAll("button")].find((button) => button.textContent === "Submit");
  assert.equal(dialog.querySelector("h2").textContent, "<img src=x onerror=alert(1)>");
  assert.equal(dialog.querySelector("h2 img"), null);
  assert.equal(image.src, "data:image/png;base64,cG5n");
  assert.equal(image.alt, "Captcha challenge");
  assert.equal(input.required, true);

  submit.click();
  assert.equal(outbox.length, 0);
  input.value = "A7Bc";
  submit.click();

  assert.equal(outbox.length, 1);
  assert.equal(outbox[0].dialogId, "captcha-1");
  assert.equal(outbox[0].value, "A7Bc");
});

test("canceling a dialog sends one null response", () => {
  setupDom();
  const outbox = [];
  const menu = new FormularMenu("root", "settings", (message) => outbox.push(message));
  const message = {
    type: "dialog.create",
    menuId: "settings",
    menuGeneration: 2,
    dialog: {
      id: "choice",
      kind: "selection",
      title: "Choose",
      options: [{ value: "a", label: "A" }]
    }
  };

  menu.feed(message);
  assert.equal(menu.feed(message), true);
  assert.equal(document.querySelectorAll("dialog").length, 1);
  const dialog = document.querySelector("dialog");
  const cancel = new window.Event("cancel", { cancelable: true });
  dialog.dispatchEvent(cancel);
  dialog.dispatchEvent(new window.Event("close"));

  assert.equal(cancel.defaultPrevented, true);
  assert.equal(outbox.length, 1);
  assert.equal(outbox[0].value, null);
});

test("ignores malformed dialog creation messages", () => {
  setupDom();
  const menu = new FormularMenu("root", "settings", () => {});

  assert.equal(menu.feed({
    type: "dialog.create",
    menuId: "settings",
    menuGeneration: 1,
    dialog: {
      id: "captcha",
      kind: "captcha",
      title: "Verify",
      resources: [{ id: "challenge", mimeType: "image/png", data: "not base64" }]
    }
  }), false);
  assert.equal(document.querySelector("dialog"), null);
});
