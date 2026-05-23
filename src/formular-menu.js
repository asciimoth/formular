const DEFAULT_PREFIX = "formular";
const STYLE_ID = "formular-menu-default-theme";

const DEFAULT_THEME = `
.formular-root{box-sizing:border-box;color:#cdd6f4;background:#1e1e2e;border:1px solid #313244;border-radius:8px;font:14px/1.45 system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;padding:14px}
.formular-root *{box-sizing:border-box}
.formular-menu{display:flex;flex-direction:column;gap:12px}
.formular-empty{color:#9399b2}
.formular-block{background:#181825;border:1px solid #313244;border-radius:8px;overflow:hidden}
.formular-block[data-inactive="true"]{opacity:.62}
.formular-block-header{align-items:center;background:#11111b;border-bottom:1px solid #313244;display:flex;gap:8px;justify-content:space-between;padding:10px 12px}
.formular-block-title{color:#f5e0dc;font-weight:700}
.formular-block-actions,.formular-form-actions,.formular-array-actions,.formular-element-actions{align-items:center;display:flex;gap:8px}
.formular-icon,.formular-button{background:#313244;border:1px solid #45475a;border-radius:6px;color:#cdd6f4;cursor:pointer;font:inherit;min-height:32px;padding:5px 10px}
.formular-icon{min-width:32px;padding:4px 8px}
.formular-icon:hover,.formular-button:hover{background:#45475a}
.formular-icon:disabled,.formular-button:disabled{cursor:not-allowed;opacity:.45}
.formular-block-body{display:flex;flex-direction:column;gap:12px;padding:12px}
.formular-header{color:#fab387;font-size:1rem;font-weight:700;margin-top:2px}
.formular-label{color:#cdd6f4}
.formular-label pre{background:#11111b;border:1px solid #313244;border-radius:6px;margin:0;overflow:auto;padding:10px}
.formular-label code,.formular-field-code{font-family:"SFMono-Regular",Consolas,"Liberation Mono",monospace}
.formular-field{display:flex;flex-direction:column;gap:5px}
.formular-field-row{align-items:center;display:flex;gap:10px;min-height:34px}
.formular-field-label{color:#bac2de;font-weight:650}
.formular-required{color:#f38ba8}
.formular-help{color:#a6adc8;font-size:.86rem}
.formular-control,.formular-select,.formular-textarea{background:#11111b;border:1px solid #45475a;border-radius:6px;color:#cdd6f4;font:inherit;min-height:34px;padding:6px 8px;width:100%}
.formular-control:focus,.formular-select:focus,.formular-textarea:focus{border-color:#89b4fa;outline:2px solid rgba(137,180,250,.25)}
.formular-textarea{min-height:88px;resize:vertical}
.formular-radio-group{display:flex;flex-wrap:wrap;gap:10px}
.formular-radio{align-items:center;display:inline-flex;gap:5px}
.formular-status{font-size:.86rem}
.formular-status[data-status="ok"]{color:#a6e3a1}
.formular-status[data-status="warn"]{color:#f9e2af}
.formular-status[data-status="error"]{color:#f38ba8}
.formular-array{border:1px dashed #45475a;border-radius:8px;padding:10px}
.formular-array-items{display:flex;flex-direction:column;gap:10px;margin-top:10px}
.formular-element{background:#11111b;border:1px solid #313244;border-radius:8px}
.formular-element-header{align-items:center;border-bottom:1px solid #313244;display:flex;justify-content:space-between;padding:8px 10px}
.formular-element-body{display:flex;flex-direction:column;gap:10px;padding:10px}
.formular-form-actions{border-top:1px solid #313244;justify-content:flex-end;padding-top:12px}
`;

function ensureDefaultTheme(prefix) {
  if (prefix !== DEFAULT_PREFIX || typeof document === "undefined" || document.getElementById(STYLE_ID)) return;
  const style = document.createElement("style");
  style.id = STYLE_ID;
  style.textContent = DEFAULT_THEME;
  document.head.append(style);
}

function clone(value) {
  if (value == null || typeof value !== "object") return value;
  return JSON.parse(JSON.stringify(value));
}

function css(prefix, name) {
  return `${prefix}-${name}`;
}

function text(value) {
  return value == null ? "" : String(value);
}

function valueKey(ref) {
  return `${ref.blockId}\n${(ref.elementPath || []).map((s) => `${s.arrayFieldId}/${s.elementId}`).join("/")}\n${ref.fieldId}`;
}

function sameField(a, b) {
  return valueKey(a) === valueKey(b);
}

function isEmpty(value) {
  return value == null || value === "" || (Array.isArray(value) && value.length === 0);
}

function normalizeKindValue(field, raw) {
  if (field.kind === "int") {
    if (raw === "") return null;
    const value = Number.parseInt(raw, 10);
    return Number.isNaN(value) ? null : value;
  }
  if (field.kind === "float" || field.kind === "range") {
    if (raw === "") return null;
    const value = Number.parseFloat(raw);
    return Number.isNaN(value) ? null : value;
  }
  if (field.kind === "checkbox") return Boolean(raw);
  return raw;
}

function renderMarkdownInline(input) {
  const root = document.createDocumentFragment();
  const parts = text(input).split(/(\[[^\]]+\]\([^)]+\)|\*\*[^*]+\*\*|`[^`]+`)/g);
  for (const part of parts) {
    if (!part) continue;
    const link = part.match(/^\[([^\]]+)\]\(([^)]+)\)$/);
    if (link) {
      const a = document.createElement("a");
      a.textContent = link[1];
      try {
        const url = new URL(link[2], window.location.href);
        if (url.protocol === "http:" || url.protocol === "https:" || url.protocol === "mailto:") a.href = url.href;
      } catch {
        a.removeAttribute("href");
      }
      a.rel = "noopener noreferrer";
      root.append(a);
      continue;
    }
    if (part.startsWith("**") && part.endsWith("**")) {
      const strong = document.createElement("strong");
      strong.textContent = part.slice(2, -2);
      root.append(strong);
      continue;
    }
    if (part.startsWith("`") && part.endsWith("`")) {
      const code = document.createElement("code");
      code.textContent = part.slice(1, -1);
      root.append(code);
      continue;
    }
    root.append(document.createTextNode(part));
  }
  return root;
}

export class FormularMenu {
  constructor(target, menuId, outbox, options = {}) {
    const node = typeof target === "string" ? document.getElementById(target) : target;
    if (!node) throw new Error("FormularMenu target node was not found");
    if (typeof outbox !== "function") throw new TypeError("FormularMenu outbox must be a function");
    this.node = node;
    this.menuId = menuId;
    this.outbox = outbox;
    this.prefix = options.classPrefix || options.prefix || DEFAULT_PREFIX;
    this.defaultTheme = options.defaultTheme !== false;
    this.blocks = new Map();
    this.menuGeneration = 0;
    this.values = new Map();
    this.focusedField = null;
    this.localElementCounter = 0;
    this.destroyed = false;
    if (this.defaultTheme) ensureDefaultTheme(this.prefix);
    this.node.classList.add(css(this.prefix, "root"));
    this.root = document.createElement("div");
    this.root.className = css(this.prefix, "menu");
    this.node.replaceChildren(this.root);
    this.render();
    this.observeDeletion();
  }

  feed(message) {
    if (this.destroyed || !message || message.menuId !== this.menuId) return false;
    if (message.type === "menu.snapshot") {
      this.menuGeneration = message.menuGeneration || 0;
      this.blocks.clear();
      this.values.clear();
      for (const block of message.blocks || []) this.blocks.set(block.id, clone(block));
      this.render();
      this.requestInitialValidation();
      return true;
    }
    if (message.type === "block.snapshot" && message.block) {
      this.blocks.set(message.block.id, clone(message.block));
      this.renderBlockById(message.block.id);
      this.requestBlockValidation(message.block);
      return true;
    }
    if (message.type === "block.delete") {
      this.blocks.delete(message.blockId);
      this.deleteBlockNode(message.blockId);
      return true;
    }
    if (message.type === "field.status") {
      this.applyFieldStatus(message);
      return true;
    }
    if (message.type === "autocomplete.hints") {
      this.applyAutocompleteHints(message);
      return true;
    }
    return false;
  }

  destroy() {
    if (this.destroyed) return;
    this.destroyed = true;
    this.observer?.disconnect();
    this.node.classList.remove(css(this.prefix, "root"));
    this.node.replaceChildren();
  }

  observeDeletion() {
    if (!this.node.parentNode || typeof MutationObserver === "undefined") return;
    this.observer = new MutationObserver(() => {
      if (!this.node.isConnected) this.destroy();
    });
    this.observer.observe(document.documentElement, { childList: true, subtree: true });
  }

  send(message) {
    if (!this.destroyed) this.outbox(message);
  }

  base(block) {
    return {
      menuId: this.menuId,
      menuGeneration: this.menuGeneration,
      blockGeneration: block?.generation || 0
    };
  }

  render() {
    if (this.destroyed) return;
    const blocks = this.sortedBlocks();
    if (blocks.length === 0) {
      const empty = document.createElement("div");
      empty.className = css(this.prefix, "empty");
      empty.textContent = "No menu snapshot received.";
      this.root.replaceChildren(empty);
      return;
    }
    this.root.replaceChildren(...blocks.map((block) => this.renderBlock(block)));
  }

  sortedBlocks() {
    return [...this.blocks.values()].sort((a, b) => (a.order - b.order) || a.id.localeCompare(b.id));
  }

  renderBlockById(blockId) {
    if (this.destroyed) return;
    const block = this.blocks.get(blockId);
    if (!block) {
      this.deleteBlockNode(blockId);
      return;
    }
    const next = this.renderBlock(block);
    const current = this.blockNode(blockId);
    if (current) {
      current.replaceWith(next);
      return;
    }
    const empty = this.root.querySelector(`.${css(this.prefix, "empty")}`);
    empty?.remove();
    const blocks = this.sortedBlocks();
    const index = blocks.findIndex((item) => item.id === blockId);
    const before = blocks.slice(index + 1).map((item) => this.blockNode(item.id)).find(Boolean);
    this.root.insertBefore(next, before || null);
  }

  deleteBlockNode(blockId) {
    this.blockNode(blockId)?.remove();
    if (this.blocks.size === 0) this.render();
  }

  blockNode(blockId) {
    return [...this.root.querySelectorAll("[data-block-id]")].find((node) => node.dataset.blockId === blockId) || null;
  }

  renderBlock(block) {
    const section = document.createElement("section");
    section.className = css(this.prefix, "block");
    section.dataset.blockId = block.id;
    section.dataset.inactive = String(Boolean(block.inactive));

    const header = document.createElement("div");
    header.className = css(this.prefix, "block-header");
    const title = document.createElement("div");
    title.className = css(this.prefix, "block-title");
    title.textContent = block.id;
    const actions = document.createElement("div");
    actions.className = css(this.prefix, "block-actions");
    if (block.copyable) actions.append(this.copyButton(block.copyable.text));
    let body;
    const collapsed = block.collapsible && block.collapsed;
    if (block.collapsible) {
      const toggle = this.button(collapsed ? "+" : "-", "Toggle block");
      toggle.addEventListener("click", () => {
        block.collapsed = !block.collapsed;
        this.render();
      });
      actions.prepend(toggle);
    }
    header.append(title, actions);
    section.append(header);
    body = document.createElement("div");
    body.className = css(this.prefix, "block-body");
    if (!collapsed) {
      for (const item of block.items || []) body.append(this.renderItem(block, item, [], block.inactive));
      if (block.form) body.append(this.renderFormActions(block));
    }
    section.append(body);
    return section;
  }

  renderItem(block, item, elementPath, disabled) {
    if (item.type === "header") {
      const node = document.createElement("div");
      node.className = css(this.prefix, "header");
      node.textContent = item.text || "";
      if (item.help) node.title = item.help;
      return node;
    }
    if (item.type === "label") return this.renderLabel(item);
    if (item.type === "button") return this.renderActionButton(block, item, elementPath, disabled);
    if (item.type === "field") return this.renderField(block, item, elementPath, disabled);
    const unknown = document.createElement("div");
    unknown.textContent = `Unsupported item: ${item.type}`;
    return unknown;
  }

  renderLabel(item) {
    const node = document.createElement("div");
    node.className = css(this.prefix, "label");
    if (item.help) node.title = item.help;
    if (item.format === "code") {
      const pre = document.createElement("pre");
      const code = document.createElement("code");
      code.textContent = item.text || "";
      pre.append(code);
      node.append(pre);
    } else if (item.format === "markdown") {
      node.append(renderMarkdownInline(item.text || ""));
    } else {
      node.textContent = item.text || "";
    }
    return node;
  }

  renderActionButton(block, item, elementPath, disabled) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = css(this.prefix, "button");
    button.textContent = item.label || item.id;
    button.disabled = Boolean(disabled || item.inactive);
    if (item.help) button.title = item.help;
    button.addEventListener("click", () => this.send({
      type: "button.press",
      ...this.base(block),
      blockId: block.id,
      elementPath: elementPath.length ? clone(elementPath) : undefined,
      buttonId: item.id
    }));
    return button;
  }

  renderField(block, field, elementPath, disabled) {
    if (field.kind === "array") return this.renderArrayField(block, field, elementPath, disabled);
    const ref = { blockId: block.id, fieldId: field.id, elementPath: elementPath.length ? clone(elementPath) : undefined };
    const current = this.getValue(ref, field.value);
    const wrapper = document.createElement("label");
    wrapper.className = css(this.prefix, "field");
    wrapper.dataset.fieldId = field.id;
    wrapper.dataset.formularFieldKey = valueKey(ref);
    const label = document.createElement("span");
    label.className = css(this.prefix, "field-label");
    label.textContent = field.label || field.id;
    if (field.required) {
      const required = document.createElement("span");
      required.className = css(this.prefix, "required");
      required.textContent = " *";
      label.append(required);
    }
    const control = this.fieldControl(block, field, ref, current, disabled);
    wrapper.append(label, control);
    if (field.help) {
      const help = document.createElement("span");
      help.className = css(this.prefix, "help");
      help.textContent = field.help;
      wrapper.append(help);
    }
    if (field.status || field.statusText) wrapper.append(this.statusNode(field.status || "unset", field.statusText || ""));
    return wrapper;
  }

  fieldControl(block, field, ref, current, disabled) {
    const readonly = disabled || field.readonly;
    if (field.kind === "checkbox") {
      const row = document.createElement("span");
      row.className = css(this.prefix, "field-row");
      const input = document.createElement("input");
      input.type = "checkbox";
      input.checked = Boolean(current);
      input.disabled = readonly;
      input.addEventListener("change", () => this.commitField(block, field, ref, input.checked));
      row.append(input);
      return row;
    }
    if (field.kind === "radio") {
      const group = document.createElement("span");
      group.className = css(this.prefix, "radio-group");
      for (const option of field.allowedValues || []) {
        const label = document.createElement("label");
        label.className = css(this.prefix, "radio");
        const input = document.createElement("input");
        input.type = "radio";
        input.name = `${this.menuId}-${valueKey(ref)}`;
        input.value = JSON.stringify(option);
        input.checked = option === current;
        input.disabled = readonly;
        input.addEventListener("change", () => input.checked && this.commitField(block, field, ref, option));
        label.append(input, document.createTextNode(text(option)));
        group.append(label);
      }
      return group;
    }
    if (field.allowedValues?.length && field.kind !== "range") {
      const select = document.createElement("select");
      select.className = css(this.prefix, "select");
      select.disabled = readonly;
      for (const option of field.allowedValues) {
        const item = document.createElement("option");
        item.value = JSON.stringify(option);
        item.textContent = text(option);
        item.selected = option === current;
        select.append(item);
      }
      select.addEventListener("change", () => this.commitField(block, field, ref, JSON.parse(select.value)));
      return select;
    }
    if (field.kind === "file") {
      const input = document.createElement("input");
      input.className = css(this.prefix, "control");
      input.type = "file";
      input.disabled = readonly;
      if (field.accept?.length) input.accept = field.accept.join(",");
      input.addEventListener("change", () => this.readFile(block, field, ref, input.files?.[0]));
      return input;
    }
    const input = field.multiline ? document.createElement("textarea") : document.createElement("input");
    input.className = field.multiline ? css(this.prefix, "textarea") : css(this.prefix, "control");
    if (!field.multiline) {
      input.type = field.secret ? "password" : field.subtype === "email" ? "email" : field.kind === "range" ? "range" : "text";
      if (field.kind === "int" || field.kind === "float") input.type = "number";
      if (field.kind === "int") input.step = "1";
      if (field.kind === "float" && field.fraction != null) input.step = String(1 / (10 ** field.fraction));
    }
    input.value = current == null ? "" : String(current);
    input.disabled = readonly;
    if (field.placeholder) input.placeholder = field.placeholder;
    if (field.min != null) input.min = String(field.min);
    if (field.max != null) input.max = String(field.max);
    input.addEventListener("input", () => {
      const value = normalizeKindValue(field, input.value);
      this.commitField(block, field, ref, value);
      if (field.autocomplete?.enabled) this.requestAutocomplete(block, field, ref, input.value);
    });
    if (field.autocomplete?.enabled) {
      const listId = `${this.prefix}-hints-${Math.random().toString(36).slice(2)}`;
      const list = document.createElement("datalist");
      list.id = listId;
      input.setAttribute("list", listId);
      input.dataset.formularFieldKey = valueKey(ref);
      input.addEventListener("focus", () => {
        this.focusedField = ref;
        this.requestAutocomplete(block, field, ref, input.value);
      });
      const group = document.createElement("span");
      group.append(input, list);
      return group;
    }
    return input;
  }

  renderArrayField(block, field, elementPath, disabled) {
    const ref = { blockId: block.id, fieldId: field.id, elementPath: elementPath.length ? clone(elementPath) : undefined };
    const elements = this.getArrayElements(ref, field);
    const wrapper = document.createElement("div");
    wrapper.className = `${css(this.prefix, "field")} ${css(this.prefix, "array")}`;
    const header = document.createElement("div");
    header.className = css(this.prefix, "field-row");
    const label = document.createElement("span");
    label.className = css(this.prefix, "field-label");
    label.textContent = field.label || field.id;
    const actions = document.createElement("span");
    actions.className = css(this.prefix, "array-actions");
    const templateSelect = document.createElement("select");
    templateSelect.className = css(this.prefix, "select");
    templateSelect.disabled = disabled || field.readonly || !(field.templates || []).length;
    for (const template of field.templates || []) {
      const option = document.createElement("option");
      option.value = template.name;
      option.textContent = template.label || template.name;
      templateSelect.append(option);
    }
    const add = this.button("+", "Add element");
    add.disabled = templateSelect.disabled;
    add.addEventListener("click", () => this.addArrayElement(block, field, ref, templateSelect.value));
    actions.append(templateSelect, add);
    header.append(label, actions);
    const items = document.createElement("div");
    items.className = css(this.prefix, "array-items");
    elements.forEach((element) => items.append(this.renderArrayElement(block, field, ref, element, disabled || field.readonly)));
    wrapper.append(header, items);
    if (field.help) {
      const help = document.createElement("div");
      help.className = css(this.prefix, "help");
      help.textContent = field.help;
      wrapper.append(help);
    }
    return wrapper;
  }

  renderArrayElement(block, arrayField, arrayRef, element, disabled) {
    const section = document.createElement("section");
    section.className = css(this.prefix, "element");
    const header = document.createElement("div");
    header.className = css(this.prefix, "element-header");
    const title = document.createElement("strong");
    title.textContent = `${arrayField.label || arrayField.id}: ${element.id}`;
    const actions = document.createElement("span");
    actions.className = css(this.prefix, "element-actions");
    if (element.copyable) actions.append(this.copyButton(element.copyable.text));
    const remove = this.button("x", "Remove element");
    remove.disabled = disabled;
    remove.addEventListener("click", () => this.removeArrayElement(block, arrayField, arrayRef, element.id));
    actions.append(remove);
    header.append(title, actions);
    const body = document.createElement("div");
    body.className = css(this.prefix, "element-body");
    const nextPath = [...(arrayRef.elementPath || []), { arrayFieldId: arrayField.id, elementId: element.id }];
    for (const item of element.items || []) body.append(this.renderItem(block, item, nextPath, disabled));
    section.append(header, body);
    return section;
  }

  renderFormActions(block) {
    const row = document.createElement("div");
    row.className = css(this.prefix, "form-actions");
    const reset = document.createElement("button");
    reset.type = "button";
    reset.className = css(this.prefix, "button");
    reset.textContent = "Reset";
    reset.disabled = block.inactive;
    reset.addEventListener("click", () => {
      this.clearBlockValues(block);
      this.render();
    });
    const apply = document.createElement("button");
    apply.type = "button";
    apply.className = css(this.prefix, "button");
    apply.dataset.formularApplyBlockId = block.id;
    apply.textContent = "Apply";
    apply.disabled = block.inactive || !this.canApply(block);
    apply.addEventListener("click", () => this.send({
      type: "form.apply",
      ...this.base(block),
      blockId: block.id,
      values: this.collectBlockValues(block)
    }));
    row.append(reset, apply);
    return row;
  }

  button(label, title) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = css(this.prefix, "icon");
    button.textContent = label;
    button.title = title;
    return button;
  }

  copyButton(copyText) {
    const button = this.button("Copy", "Copy");
    button.addEventListener("click", async () => {
      try {
        await navigator.clipboard?.writeText(copyText || "");
      } catch {
        this.send({ type: "clipboard.copy.failed", menuId: this.menuId });
      }
    });
    return button;
  }

  statusNode(status, statusText) {
    const node = document.createElement("span");
    node.className = css(this.prefix, "status");
    node.dataset.status = status;
    node.textContent = statusText || status;
    return node;
  }

  getValue(ref, fallback) {
    const key = valueKey(ref);
    if (!this.values.has(key)) this.values.set(key, clone(fallback ?? null));
    return this.values.get(key);
  }

  setValue(block, field, ref, value) {
    this.values.set(valueKey(ref), value);
    if (ref.elementPath?.length) this.syncNestedValue(ref, value);
    if (field.kind !== "array") return;
    this.values.set(valueKey(ref), value);
  }

  commitField(block, field, ref, value) {
    this.setValue(block, field, ref, value);
    field.value = value;
    if (field.validate && field.kind !== "file") {
      field.status = "unset";
      field.statusText = "";
      this.updateFieldStatusDOM(ref, field);
    }
    if (field.validate && field.kind !== "file") this.send({ type: "field.validate", ...this.base(block), field: ref, value });
    if (!block.form) this.send({ type: "field.update", ...this.base(block), field: ref, value });
    if (block.form) this.updateFormActions(block);
  }

  readFile(block, field, ref, file) {
    if (!file) return;
    const maxBytes = field.maxBytes || 4098;
    if (file.size > maxBytes) {
      this.applyLocalStatus(ref, "error", `File is larger than ${maxBytes} bytes`);
      return;
    }
    const reader = new FileReader();
    reader.addEventListener("load", () => {
      const value = String(reader.result || "").split(",", 2)[1] || "";
      this.commitField(block, field, ref, value);
    });
    reader.readAsDataURL(file);
  }

  requestAutocomplete(block, field, ref, prefix) {
    this.focusedField = ref;
    this.send({ type: "autocomplete.request", ...this.base(block), field: ref, prefix });
  }

  requestInitialValidation() {
    for (const block of this.blocks.values()) this.requestBlockValidation(block);
  }

  requestBlockValidation(block) {
    for (const item of block.items || []) this.requestItemValidation(block, item, []);
  }

  requestItemValidation(block, item, elementPath) {
    if (item.type !== "field") return;
    const ref = { blockId: block.id, fieldId: item.id, elementPath: elementPath.length ? clone(elementPath) : undefined };
    if (item.kind === "array") {
      const elements = this.getArrayElements(ref, item);
      for (const element of elements || []) {
        const nextPath = [...elementPath, { arrayFieldId: item.id, elementId: element.id }];
        for (const child of element.items || []) this.requestItemValidation(block, child, nextPath);
      }
      return;
    }
    if (!item.validate || item.kind === "file") return;
    this.send({ type: "field.validate", ...this.base(block), field: ref, value: this.getValue(ref, item.value) });
  }

  applyAutocompleteHints(message) {
    if (!this.focusedField || !sameField(this.focusedField, message.field)) return;
    const key = valueKey(message.field);
    const input = [...this.root.querySelectorAll("[data-formular-field-key]")].find((node) => node.dataset.formularFieldKey === key);
    const list = input ? document.getElementById(input.getAttribute("list")) : null;
    if (!input || !list || input.value !== message.prefix) return;
    list.replaceChildren(...(message.hints || []).filter((hint) => String(hint).startsWith(message.prefix)).map((hint) => {
      const option = document.createElement("option");
      option.value = hint;
      return option;
    }));
  }

  applyFieldStatus(message) {
    const field = this.findField(message.field);
    if (!field) return;
    field.status = message.status;
    field.statusText = message.statusText || "";
    if (message.readonly != null) field.readonly = Boolean(message.readonly);
    this.updateFieldStatusDOM(message.field, field);
    const block = this.blocks.get(message.field.blockId);
    if (block?.form) this.updateFormActions(block);
  }

  applyLocalStatus(ref, status, statusText) {
    const field = this.findField(ref);
    if (field) {
      field.status = status;
      field.statusText = statusText;
      this.updateFieldStatusDOM(ref, field);
      const block = this.blocks.get(ref.blockId);
      if (block?.form) this.updateFormActions(block);
    }
  }

  fieldNode(ref) {
    const key = valueKey(ref);
    return [...this.root.querySelectorAll("[data-formular-field-key]")].find((node) => node.dataset.formularFieldKey === key) || null;
  }

  updateFieldStatusDOM(ref, field) {
    const node = this.fieldNode(ref);
    if (!node) return;
    let status = node.querySelector(`.${css(this.prefix, "status")}`);
    if (!field.status && !field.statusText) {
      status?.remove();
    } else {
      if (!status) {
        status = this.statusNode(field.status || "unset", field.statusText || "");
        node.append(status);
      }
      status.dataset.status = field.status || "unset";
      status.textContent = field.statusText || field.status || "unset";
    }
    if (field.readonly != null) {
      for (const control of node.querySelectorAll("input, select, textarea")) {
        control.disabled = Boolean(field.readonly);
      }
    }
  }

  updateFormActions(block) {
    const apply = [...this.root.querySelectorAll("[data-formular-apply-block-id]")]
      .find((button) => button.dataset.formularApplyBlockId === block.id);
    if (apply) apply.disabled = block.inactive || !this.canApply(block);
  }

  findField(ref) {
    const block = this.blocks.get(ref.blockId);
    if (!block) return null;
    let items = block.items || [];
    for (const segment of ref.elementPath || []) {
      const array = items.find((item) => item.type === "field" && item.id === segment.arrayFieldId);
      const element = array?.elements?.find((item) => item.id === segment.elementId);
      items = element?.items || [];
    }
    return items.find((item) => item.type === "field" && item.id === ref.fieldId) || null;
  }

  getArrayElements(ref, field) {
    const key = valueKey(ref);
    if (!this.values.has(key)) this.values.set(key, clone(field.elements || []));
    return this.values.get(key);
  }

  addArrayElement(block, field, ref, templateName) {
    const template = (field.templates || []).find((item) => item.name === templateName);
    if (!template) return;
    const elements = this.getArrayElements(ref, field);
    const element = {
      id: `local-${++this.localElementCounter}`,
      template: template.name,
      items: clone(template.items || [])
    };
    elements.push(element);
    this.values.set(valueKey(ref), elements);
    const value = this.arrayWireValues(elements);
    if (field.validate) this.send({ type: "field.validate", ...this.base(block), field: ref, value });
    if (!block.form) this.send({ type: "field.update", ...this.base(block), field: ref, value });
    this.render();
  }

  removeArrayElement(block, field, ref, elementId) {
    const elements = this.getArrayElements(ref, field).filter((element) => element.id !== elementId);
    this.values.set(valueKey(ref), elements);
    const value = this.arrayWireValues(elements);
    if (field.validate) this.send({ type: "field.validate", ...this.base(block), field: ref, value });
    if (!block.form) this.send({ type: "field.update", ...this.base(block), field: ref, value });
    this.render();
  }

  syncNestedValue(ref, value) {
    const last = ref.elementPath[ref.elementPath.length - 1];
    const parentRef = {
      blockId: ref.blockId,
      fieldId: last.arrayFieldId,
      elementPath: ref.elementPath.slice(0, -1)
    };
    const elements = this.values.get(valueKey(parentRef));
    const element = elements?.find((item) => item.id === last.elementId);
    const field = element?.items?.find((item) => item.type === "field" && item.id === ref.fieldId);
    if (field) field.value = value;
  }

  arrayWireValues(elements) {
    return (elements || []).map((element) => ({
      id: element.id,
      template: element.template,
      values: this.collectItemValues(element.items || [])
    }));
  }

  collectItemValues(items) {
    const values = {};
    for (const item of items || []) {
      if (item.type !== "field") continue;
      values[item.id] = item.kind === "array" ? this.arrayWireValues(item.elements || []) : clone(item.value ?? null);
    }
    return values;
  }

  collectBlockValues(block) {
    const values = {};
    for (const item of block.items || []) {
      if (item.type !== "field") continue;
      const ref = { blockId: block.id, fieldId: item.id };
      const value = this.getValue(ref, item.kind === "array" ? item.elements || [] : item.value);
      values[item.id] = item.kind === "array" ? this.arrayWireValues(value) : clone(value);
    }
    return values;
  }

  clearBlockValues(block) {
    for (const key of [...this.values.keys()]) {
      if (key.startsWith(`${block.id}\n`)) this.values.delete(key);
    }
  }

  canApply(block) {
    for (const item of block.items || []) {
      if (item.type !== "field") continue;
      const value = this.getValue({ blockId: block.id, fieldId: item.id }, item.value);
      if (item.required && isEmpty(value)) return false;
      if (item.validate && item.kind !== "file" && item.status !== "ok") return false;
    }
    return true;
  }
}

if (typeof window !== "undefined") window.FormularMenu = FormularMenu;

export default FormularMenu;
