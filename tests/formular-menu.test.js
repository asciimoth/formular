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
