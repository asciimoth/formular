# formular

Formular is a JSON message DSL for backend-owned dynamic menus and forms that
can be rendered by GUI, TUI, CLI, MCP, web, or other frontends.

- Protocol and implementation notes: [docs/protocol.md](docs/protocol.md)
- Browser frontend library: [docs/frontend-js.md](docs/frontend-js.md)
- JSON Schemas: [schemas/](schemas/)
- Go wire types: [messages.go](messages.go)

## JavaScript frontend

The npm package entry point is a single dependency-free browser file:

```js
import { FormularMenu } from "@asciimoth/formular-menu";

const menu = new FormularMenu("root", "settings", (message) => {
  console.log("frontend message", message);
});
```

Run the local WASM demo:

```sh
pnpm install
pnpm run demo
```

The demo renders two independent frontends and a Go WASM backend connected by
one merged JSON message channel.
