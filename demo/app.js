import { FormularMenu } from "../src/formular-menu.js";

const logNode = document.getElementById("message-log");
const stateNode = document.getElementById("backend-state");
const restartFrontendsButton = document.getElementById("restart-frontends");
const restartBackendButton = document.getElementById("restart-backend");
const frontends = [];
let backendGeneration = 0;

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
  frontends.push(new FormularMenu("left-menu", "left", sendToBackend));
  frontends.push(new FormularMenu("right-menu", "right", sendToBackend));
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
