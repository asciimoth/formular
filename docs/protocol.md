# Formular Protocol

Formular is a small JSON message DSL for dynamic menus and forms.
It is meant for systems where the backend owns menu logic, while a frontend
renders the current menu in a GUI, TUI, CLI, MCP client, web page, or another
non-HTML-specific surface.

The protocol deliberately avoids styling and complex layout.
A frontend can choose its own visual presentation, but it should preserve the
structure, labels, validation state, enabled state, and field values sent by
the backend.

## Transport Model
Messages are bidirectional JSON values. The transport is not part of this
specification; WebSocket, HTTP long polling, Unix sockets, stdio,
in-process calls, and similar channels are all valid.

The protocol uses push semantics: either side may send a valid message whenever
it has new state.
Request/response transports can emulate this with e.g. long polling.

Every message must contain:
- `type`: message kind.
- `menuId`: application-defined menu instance ID.
- `menuGeneration`: backend-assigned menu structure generation when known.
- `blockGeneration`: backend-assigned block structure generation for block-scoped messages.

Frontends should ignore messages for unknown or currently hidden menus.
Both sides should validate message schema and values before acting on a message.

## Initialization
Communication starts when the backend sends a `menu.snapshot` message to the frontend. A specific application may let the frontend request that snapshot first, but that request is outside this protocol.

## Schemas
JSON Schemas live in `schemas/`.
- `common.schema.json`: shared block, item, field, value, and reference definitions.
- `backend-message.schema.json`: union of all backend-to-frontend messages.
- `frontend-message.schema.json`: union of all frontend-to-backend messages.
- `*-message.schema.json`: concrete message schemas.

The schemas use JSON Schema draft 2020-12.

## Menu Model
A menu contains zero or more blocks. 
A menu with no blocks is valid but not useful.

Blocks are independently replaceable snapshots. Each block has:
- `id`: unique string within the menu.
- `order`: integer sort key for frontend display. It is part of the API model, not user-facing text.
- `generation`: backend-assigned block generation.
- `form`: whether the block is applied as a whole.
- `inactive`: whether all user input in the block is disabled.
- `collapsible`: whether the frontend should expose local collapse controls.
- `collapsed`: initial collapse state.
- `copyable`: optional clipboard text for an explicit frontend copy action.
- `items`: headers, labels, progress bars, logs, buttons, and fields.

The backend increments `menuGeneration` when blocks are added or removed.
It increments a block `generation` when fields are added or removed,
field configuration changes, or the block snapshot changes structurally.

Form blocks are applied as a whole.
The frontend should render frontend-owned Apply and Reset controls for them;
these controls are not declared by the backend as normal buttons.
When the user applies a form block, the frontend sends one `form.apply` message
containing all field values in the block.

Non-form blocks send individual `field.update` messages as the user edits nput fields.
Frontends may debounce these updates to avoid sending excessive traffic.

Block `form` status should not change in place.
Delete and recreate the block if that mode changes.

Collapse state is local frontend state.
A frontend should not send collapse changes to the backend,
and should ignore backend changes to `collapsed` except in a forced `menu.snapshot`.

## Items
Each item has a `type`.
- `header`: plain section heading. Headers have no levels and are not allowed inside array elements.
- `label`: display text. `format` can be `plain`, `markdown`, or `code`; `syntax` may name a code highlighting language.
- `progressbar`: readonly progress display with `label` and integer `progress` percentage from 0 to 100.
- `logs`: readonly list of log lines with a colored `level` prefix. Levels are `trace`, `debug`, `info`, `warn`, `error`, and `panic`.
- `button`: user action. Inactive buttons must not be triggerable.
- `field`: user input.

All items may have plaintext `help`.
Frontends can show help as a tooltip, side panel, popover, or equivalent.

Headers have no levels because Formular menus do not define nested sections.
Plain labels are unformatted by default.
Markdown labels intentionally do not define a strict markdown subset, but frontends should support basic links and text styling when practical.
Code labels should be displayed as monospace text; syntax highlighting is optional and controlled by `syntax`.

Progress bars are display-only items. Frontends should render `progress` as a bounded percentage and must not send frontend messages when it changes.

Logs are display-only items. Each log line has a `level` and `text`; frontends should prepend the level before the text and visually distinguish levels when possible.

Buttons can appear above, below, or between fields according to item order.
When a button is activated, the frontend sends `button.press`.

## Fields
Base field kinds are:
- `text`: string input.
- `int`: integer input.
- `float`: floating point input.
- `file`: file upload encoded as base64 text.
- `checkbox`: boolean input.
- `radio`: one value selected from `allowedValues`.
- `range`: numeric range input.
- `array`: ordered collection of templated elements.

Common field properties:
- `id`: unique string within the owning block or array element.
- `kind`: field kind. This should not be changed in place; replace the block snapshot instead.
- `label`: user-facing field caption.
- `value`: backend-provided current or default value.
- `placeholder`: optional empty-state text for text-like inputs.
- `help`: optional plaintext hint.
- `readonly`: frontend displays the value but prevents editing.
- `required`: frontend prevents form apply while empty.
- `validate`: frontend sends `field.validate` on changes.
- `copyable`: optional clipboard text for an explicit frontend copy action.
- `status`: backend state, one of `unset`, `ok`, `warn`, `error`.
- `statusText`: optional explanation.

The backend can update `readonly`, `status`, and `statusText` with `field.status`.
Frontends should use the same visual language for frontend-side validation and backend-supplied `ok`, `warn`, and `error` states.

Fields with `validate: true` require backend validation on user input.
A frontend should not allow form apply while a validation-enabled field has not been marked `ok` by the backend.
File fields are an exception and are excluded from realtime validation.

Required fields block form apply while empty.

### Text Fields
`text` fields use string values.

Supported text-specific properties:
- `secret`: hides entered text in the frontend, for example with bullets.
- `autocomplete`: optional object with `enabled` and arbitrary application-defined `tag`.
- `placeholder`: empty-state text.
- `multiline`: whether newlines are allowed.
- `allowedValues`: optional predefined values a frontend may expose as choices when they do not conflict with the subtype.
- `subtype`: optional subtype. Unknown subtypes should be treated as generic text.

Known text subtypes:
- `email`: always single-line.
- `filepath`: always single-line. A capable frontend should provide a file picker for choosing a path. This is still a path string input, not file upload. Values may use Unix or Windows path syntax.

For single-line text fields, frontends should ignore or join newline characters before sending values.

### Integer Fields
`int` fields use integer values.

Supported integer-specific properties:
- `min`: optional minimum.
- `max`: optional maximum.
- `allowedValues`: optional predefined integer values a frontend may expose as choices.

### Float Fields
`float` fields use JSON number values.

Supported float-specific properties:
- `min`: optional minimum.
- `max`: optional maximum.
- `fraction`: optional number of fractional digits.
- `allowedValues`: optional predefined number values a frontend may expose as choices.

### File Fields
`file` fields upload file content as base64 text in the JSON `value`.

Supported file-specific properties:
- `maxBytes`: maximum raw file size before base64 encoding. The default is 4098 bytes.
- `accept`: optional list of accepted MIME types.

Frontends may ignore `accept` if they cannot check MIME types. Frontends should ignore `validate` on file fields because file fields are excluded from realtime validation.

### Checkbox Fields
`checkbox` fields use boolean values.

They represent a single true/false input. `required` on a checkbox means the application expects it to be filled according to its own validation rules; backends should still validate submitted values.

### Radio Fields
`radio` fields use one selected value. They should normally define `allowedValues`; frontends should render one choice from that list.

### Range Fields
`range` fields use numeric values.

Supported range-specific properties:
- `min`: optional lower bound.
- `max`: optional upper bound.

A frontend may render a slider, stepper, numeric input, or another range-appropriate control.

### Array Field Values
`array` fields use an array of element values. Their structure is described in the next section.

## Array Fields
Array fields contain element templates and current elements.

Templates have names unique within the array field. Elements declare the template they are based on. Element content is block-like but cannot contain headers, nested array fields, or form behavior.

Frontends should provide controls for adding and removing elements. If an array field has multiple templates, adding an element should include template selection.

Only the whole containing block can be a form. Array elements are never forms by themselves.

Array field values are sent as arrays of objects:

```json
{
  "id": "db-1",
  "template": "database",
  "values": {
    "host": "localhost",
    "port": 5432
  }
}
```

## Copyable Objects
Blocks, fields, and array elements may include:

```json
{ "copyable": { "text": "value for clipboard" } }
```

Frontends should expose an explicit copy action that places this text on the system clipboard. This is separate from ordinary user text selection and copy behavior.

Whole blocks may be marked inactive. Frontends should visually mark inactive blocks as inaccessible and prevent any user input inside them.

## Autocompletion
Autocompletion is optional for both backends and frontends.

For fields with `autocomplete.enabled: true`, the frontend may send `autocomplete.request` messages with the current input state. The backend may send `autocomplete.hints` messages with suggested complete values at any time, though it most commonly does so in response to a request.

Frontends may use autocomplete hints while the user types. They should ignore autocomplete hints for fields that are not currently focused. They should also ignore suggested values that do not have the current input state as a prefix.

## Backend Messages
Backend-to-frontend messages are snapshots of their subject, not mutation diffs. The frontend compares the received snapshot with local view state and applies whatever UI changes are necessary.

`menu.snapshot`

Initializes or replaces the full menu. With `force: true`, the frontend must discard all remembered menu and block state, including collapse state and known generations.

`block.snapshot`

Creates or replaces one block with a full block snapshot.

`block.delete`

Deletes one block by `blockId`. Block list changes should be honored even when generation handling is otherwise stale.

`field.status`

Updates validation state, status text, or readonly flag for one field. Frontends should ignore stale status updates when their generation is older than the current block generation.

`autocomplete.hints`

Provides complete candidate values for a focused text field. Frontends should ignore hints for fields that are not currently focused and hints whose values do not start with the current input prefix.

## Frontend Messages
Frontend-to-backend messages describe user input or frontend requests for the active menu. Backends should ignore messages for unknown menu IDs, stale menus they no longer serve, malformed messages, or messages that fail semantic validation.

`field.update`

Sent for realtime updates in non-form blocks. Frontends may debounce this message to avoid flooding the backend.

`field.validate`

Sent for fields with `validate: true` after user input changes. Frontends should block form apply while required validation-enabled fields have not been marked `ok` by the backend.

`form.apply`

Sent when a user applies a form block. It contains the complete current value map for all fields in the block, including array fields.

`button.press`

Sent when a declared button is activated. For buttons inside array elements, `elementPath` identifies the containing element.

`autocomplete.request`

Optional request sent for focused autocomplete-enabled fields. The backend may answer with `autocomplete.hints`.

## Frontend Behavior
Frontends should:

- Use a vertical layout by default unless their platform has a better native convention.
- Render blocks sorted by `order`, then by `id`.
- Render form blocks with frontend-owned Apply and Reset controls.
- Keep collapse state local unless a forced menu snapshot arrives.
- Ignore backend messages without `menuId` or with an unknown `menuId`.
- Ignore autocomplete hints that are unrelated to the currently focused input.
- Prevent user input in inactive blocks, readonly fields, and inactive buttons.
- Prevent form apply when required fields are empty.
- Prevent form apply when validation-enabled fields have not received backend `ok`.
- Avoid interrupting active user input while applying backend updates unless the edited object is deleted or force-replaced.
- Ignore malformed messages instead of crashing; logging is recommended.

Markdown labels should support basic text styling and links when possible. Code labels should be shown in monospace even when syntax highlighting is unavailable.

## Backend Behavior
Backends should:

- Send snapshots, not mutation diffs.
- Treat frontend messages as untrusted input.
- Validate all field values, array elements, and array element templates.
- Increment menu and block generations consistently.
- Prefer smaller independently updated blocks over one large block when state changes frequently.
- Resend `menu.snapshot` with `force: true` when frontend state must be reset.

The backend is authoritative in conflicts. Multiple frontends may interact with the same menu concurrently.

## Race Conditions
Generation numbers allow receivers to identify stale messages.

When a backend message changes block membership or array element membership inside a block, the frontend should honor it even if the local state appears newer. When a backend message only changes field flags or validation marks, the frontend should ignore it if the referenced block generation is older than the local block generation.

Frontend messages include the generations the frontend saw when the user acted. The backend may reject, ignore, or reconcile stale frontend actions according to application policy.

## Security
Neither side should trust the other. Validate schemas and semantic constraints.

Backends must validate array field values against their declared templates. Web frontends must sanitize markdown, links, code labels, status text, help text, and any other backend-provided text before inserting it into the DOM. File payloads should be size-limited before decoding and should not be executed or trusted based on client-provided MIME type.

## Minimal Example
```json
{
  "type": "menu.snapshot",
  "menuId": "settings",
  "menuGeneration": 1,
  "blocks": [
    {
      "id": "account",
      "order": 10,
      "generation": 1,
      "form": true,
      "items": [
        { "type": "header", "id": "title", "text": "Account" },
        { "type": "progressbar", "id": "sync", "label": "Sync", "progress": 40 },
        {
          "type": "logs",
          "id": "events",
          "label": "Events",
          "logs": [
            { "level": "info", "text": "Profile opened" }
          ]
        },
        {
          "type": "field",
          "id": "email",
          "kind": "text",
          "label": "Email",
          "subtype": "email",
          "required": true,
          "validate": true
        }
      ]
    }
  ]
}
```

```json
{
  "type": "field.validate",
  "menuId": "settings",
  "menuGeneration": 1,
  "blockGeneration": 1,
  "field": { "blockId": "account", "fieldId": "email" },
  "value": "user@example.com"
}
```
