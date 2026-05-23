# FormularMenu JavaScript frontend

`src/formular-menu.js` is a dependency-free browser library for rendering
Formular protocol menus into an existing DOM node.

## Constructor

```js
import { FormularMenu } from "@asciimoth/formular-menu";

const menu = new FormularMenu("settings-root", "settings", (message) => {
  websocket.send(JSON.stringify(message));
});
```

Signature:

```ts
new FormularMenu(target, menuId, outbox, options?)
```

- `target`: an `HTMLElement` or the id of an existing DOM element.
- `menuId`: the Formular protocol `menuId` this frontend instance owns.
- `outbox`: callback receiving frontend-to-backend protocol messages.
- `options.classPrefix`: CSS class prefix. Defaults to `formular`.
- `options.defaultTheme`: set to `false` to skip the built-in Catppuccin theme.

The constructor clears the target node and owns its contents until `destroy()`
is called or the target node is removed from the document.

## Methods

```ts
menu.feed(message): boolean
```

Applies one backend-to-frontend message. It returns `true` when the message was
for this menu and was handled. Unknown message types, missing messages, and
messages for other menu IDs are ignored.

```ts
menu.destroy(): void
```

Disconnects observers, removes owned DOM, and stops outgoing messages.

## Styling

By default the library injects a Catppuccin-themed stylesheet for the
`formular-*` class prefix. For application-owned styles, pass a different
prefix and provide matching CSS:

```js
new FormularMenu(root, "settings", send, {
  classPrefix: "my-menu",
  defaultTheme: false
});
```

The renderer emits stable classes such as `PREFIX-root`, `PREFIX-block`,
`PREFIX-field`, `PREFIX-control`, `PREFIX-button`, and `PREFIX-status`.

## Message behavior

- `menu.snapshot` creates or replaces the full rendered menu.
- `block.snapshot` creates or replaces one block.
- `block.delete` removes one block.
- `field.status` updates validation state, status text, and readonly state.
- `autocomplete.hints` populates the focused text input datalist.
- Non-form field edits send `field.update`.
- Fields with `validate: true` send `field.validate`.
- Form blocks render local Reset and Apply controls; Apply sends `form.apply`.
- Buttons send `button.press`.
- Autocomplete-enabled text inputs send `autocomplete.request`.

Array fields are edited locally and serialized as the protocol's
`ArrayElementValue[]` shape when sent to the backend.
