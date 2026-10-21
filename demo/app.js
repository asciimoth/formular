import { FormularMenu } from "../src/formular-menu.js";

const logNode = document.getElementById("message-log");
const stateNode = document.getElementById("backend-state");
const restartFrontendsButton = document.getElementById("restart-frontends");
const restartBackendButton = document.getElementById("restart-backend");
const frontends = [];
const frontendKnownUsers = ["Ada", "Grace", "Linus"];
let backendGeneration = 0;

// This state belongs to the frontend. FormularMenu requests the current values
// again whenever the user opens a field marked with selector: "users".
window.formularDemoKnownUsers = frontendKnownUsers;

function selectorValues(selector) {
  return selector === "users" ? frontendKnownUsers : undefined;
}

function highlightJSON(code, source) {
  const pattern = /"(?:\\.|[^"\\])*"|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|\b(?:true|false|null)\b/g;
  const nodes = [];
  let offset = 0;
  for (const match of source.matchAll(pattern)) {
    if (match.index > offset) nodes.push(document.createTextNode(source.slice(offset, match.index)));
    const token = document.createElement("span");
    const rest = source.slice(match.index + match[0].length);
    if (match[0].startsWith("\"")) token.className = /^\s*:/.test(rest) ? "demo-json-key" : "demo-json-string";
    else if (/^(?:true|false|null)$/.test(match[0])) token.className = "demo-json-literal";
    else token.className = "demo-json-number";
    token.textContent = match[0];
    nodes.push(token);
    offset = match.index + match[0].length;
  }
  nodes.push(document.createTextNode(source.slice(offset) || (source.endsWith("\n") ? " " : "")));
  code.replaceChildren(...nodes);
}

function renderDemoMultilineTextInput(textarea, { field, onValueChange }) {
  if (field.subtype !== "json") return undefined;

  const editor = document.createElement("span");
  editor.className = "demo-highlight-editor";
  const highlight = document.createElement("pre");
  highlight.className = "demo-highlight-layer";
  highlight.setAttribute("aria-hidden", "true");
  const code = document.createElement("code");
  highlight.append(code);
  const update = () => highlightJSON(code, textarea.value);
  const syncScroll = () => {
    highlight.scrollTop = textarea.scrollTop;
    highlight.scrollLeft = textarea.scrollLeft;
  };
  textarea.addEventListener("input", update);
  textarea.addEventListener("scroll", syncScroll);
  onValueChange(update);
  editor.append(highlight, textarea);
  return editor;
}

function log(direction, message) {
  const item = document.createElement("li");
  item.textContent = `${direction} ${message.type} (${message.menuId})`;
  logNode.prepend(item);
  while (logNode.children.length > 16) logNode.lastElementChild.remove();
}

function sendToBackend(message) {
  log("frontend -> middleware", message);
  window.formularMiddlewareSend?.(JSON.stringify(message));
}

function createFrontends({ requestSnapshot = true } = {}) {
  for (const frontend of frontends) frontend.destroy();
  frontends.length = 0;
  const options = {
    selectors: selectorValues,
    renderMultilineTextInput: renderDemoMultilineTextInput
  };
  frontends.push(new FormularMenu("left-menu", "left", sendToBackend, options));
  frontends.push(new FormularMenu("right-menu", "right", sendToBackend, options));
  log("demo", { type: "demo.frontends.started", menuId: "both" });
  if (requestSnapshot) requestSnapshots();
}

function requestSnapshots() {
  if (typeof window.formularMiddlewareSend !== "function") return;
  sendToBackend({ type: "demo.snapshot.request", menuId: "demo" });
}

window.formularFrontendDispatch = (raw) => {
  let message;
  try {
    message = JSON.parse(raw);
  } catch (error) {
    stateNode.textContent = `Ignored malformed backend message: ${error.message}`;
    return;
  }
  if (!message || typeof message !== "object" || typeof message.type !== "string") {
    stateNode.textContent = "Ignored invalid backend message";
    return;
  }
  log("backend -> frontends", message);
  for (const frontend of frontends) frontend.feed(message);
};

async function bootBackend() {
  backendGeneration += 1;
  const generation = backendGeneration;
  stateNode.textContent = `Starting Go backend #${generation}`;
  const go = new Go();
  const response = await fetch("./public/formular-demo.wasm");
  const result = await WebAssembly.instantiateStreaming(response, go.importObject);
  stateNode.textContent = `Go backend #${generation} running`;
  log("demo", { type: "demo.backend.started", menuId: `backend-${generation}` });
  go.run(result.instance).catch((error) => {
    stateNode.textContent = `Backend #${generation} failed: ${error.message}`;
    console.error(error);
  });
}

restartFrontendsButton.addEventListener("click", () => createFrontends());
restartBackendButton.addEventListener("click", () => {
  bootBackend().catch((error) => {
    stateNode.textContent = `Backend failed: ${error.message}`;
    console.error(error);
  });
});

createFrontends({ requestSnapshot: false });
bootBackend().catch((error) => {
  stateNode.textContent = `Backend failed: ${error.message}`;
  console.error(error);
});
