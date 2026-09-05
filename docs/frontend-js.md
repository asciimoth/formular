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
- `options.selectors`: callback that returns current frontend-owned string
  values for a selector name.

For example, the backend can mark a text field with `"selector": "users"`
without sending a user list. Supply that list when you create the frontend:

```js
const menu = new FormularMenu(root, "settings", send, {
  selectors: (selector) => selector === "users" ? currentUserNames() : undefined
});
```

The callback must return an array of strings, or `null` or `undefined` for an
unknown selector. The frontend calls it during rendering and again before each
pointer click opens the selection control. Thus, the control shows the current
frontend values. If the callback does not know the requested name, the frontend
renders a normal text input. If the current field value is not in the frontend
list, the control also shows that value so its displayed and submitted values
stay consistent.

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
`PREFIX-field`, `PREFIX-control`, `PREFIX-progressbar`, `PREFIX-logs`,
`PREFIX-button`, and `PREFIX-status`.
Text-like `PREFIX-control` and `PREFIX-textarea` fields also receive a
`data-status` attribute when the backend supplies a field status.

## Message behavior

- `menu.snapshot` creates or replaces the full rendered menu.
- `block.snapshot` creates or replaces one block.
- `block.delete` removes one block.
- `field.status` updates validation state, status text, and readonly state.
- `autocomplete.hints` populates the focused text input datalist.
- `dialog.create` creates a native `<dialog>` and opens it with `showModal()`.
- `progressbar` items render readonly progress and do not send frontend messages.
- `logs` items render readonly log lines with colored level prefixes.
- Non-form field edits send `field.update`.
- Fields with `validate: true` send `field.validate`.
- Form blocks render local Reset and Apply controls; Apply sends `form.apply`.
- Buttons send `button.press`.
- Autocomplete-enabled text inputs send `autocomplete.request`.
- A completed or canceled dialog sends one `dialog.response`.

Array fields are edited locally and serialized as the protocol's
`ArrayElementValue[]` shape when sent to the backend.

## Frontend item state

The renderer supports the protocol `stateConditions` property on every item.
It evaluates `visible` and `readonly` conditions from current local field
values. It evaluates them when it receives a snapshot, when a field changes,
and when a form resets. This evaluation does not send a message.

For example:

```js
menu.feed({
  type: "menu.snapshot",
  menuId: "settings",
  menuGeneration: 1,
  blocks: [{
    id: "advanced",
    order: 1,
    generation: 1,
    form: true,
    items: [
      { type: "field", id: "enabled", kind: "checkbox", label: "Enabled", value: false },
      {
        type: "field",
        id: "details",
        kind: "text",
        label: "Details",
        stateConditions: {
          visible: { source: { fieldId: "enabled" }, operator: "equals", value: true }
        }
      }
    ]
  }]
});
```

The renderer updates existing DOM nodes in place. A source text control keeps
focus while its value changes the state of another item. It uses the native
`hidden` property for visibility and the native `disabled` property for
readonly controls. The item nodes also have `data-formular-readonly` for custom
styles and tests.

Relative field sources work inside array elements. Set `source.blockId` to use
a top-level source field in another block. See [the protocol
specification](protocol.md#frontend-item-state-conditions) for operators,
Boolean composition, missing-source behavior, and form submission rules.

## Dialogs

The browser frontend supports `yesno`, `selection`, and `captcha` dialogs.
Dialogs are independent of blocks. A dialog can arrive before or after a menu
snapshot.

```js
menu.feed({
  type: "dialog.create",
  menuId: "settings",
  menuGeneration: 4,
  dialog: {
    id: "delete-profile",
    kind: "yesno",
    title: "Delete profile?",
    text: "This action cannot be undone."
  }
});
```

The renderer appends a `<dialog>` to the owned target and calls `showModal()`.
Yes and No send boolean values. Selection sends a string or string array.
Captcha sends the text input. Escape and Cancel send `null`.
After the first response, the renderer closes and removes the element.

For captcha images, send standard base64 text. Do not send a remote URL:

```js
{
  type: "dialog.create",
  menuId: "settings",
  menuGeneration: 4,
  dialog: {
    id: "captcha",
    kind: "captcha",
    title: "Verification",
    resources: [{
      id: "challenge",
      mimeType: "image/png",
      data: "iVBORw0KGgoAAAANSUhEUgAA...",
      alt: "Captcha challenge"
    }]
  }
}
```

The default theme uses `PREFIX-dialog`, `PREFIX-dialog-form`,
`PREFIX-dialog-title`, `PREFIX-dialog-text`, `PREFIX-dialog-resources`,
`PREFIX-dialog-resource`, `PREFIX-dialog-resource-image`, and
`PREFIX-dialog-actions`. The renderer uses safe text DOM properties for dialog
titles, text, labels, and alternative text.
