# Using Formular from a backend
Formular is a JSON message protocol for backend-owned menus and forms.
The backend describes the current menu state, the frontend renders it, and user
actions come back as protocol messages.
This guide focuses only on backend-side menu logic: which messages to send,
which messages to handle, and what value shapes to expect.

It intentionally does not cover project setup, transports,
JSON marshaling/unmarshaling, WebSocket plumbing, or frontend implementation details.

The examples use JSON first.
Small Go snippets are included only to show how the same shape maps to the
[asciimoth/formular](github.com/asciimoth/formular) wire types.

## Snapshots, not patches
Backend-to-frontend messages are snapshots of a menu, a block, a field status,
or autocomplete hints.
Do not send small mutation diffs such as “rename this field” or “append this item”.
Instead, rebuild the affected menu or block and send the whole snapshot for that
subject.

The usual backend message types are:

| Message              | Purpose                                               |
| -------------------- | ----------------------------------------------------- |
| `menu.snapshot`      | Initialize or replace the whole menu.                 |
| `block.snapshot`     | Create or replace one block.                          |
| `block.delete`       | Remove one block.                                     |
| `field.status`       | Update field validation status and/or readonly state. |
| `autocomplete.hints` | Send possible completions for a focused text field.   |

The usual frontend message types are:

| Message                | Purpose                                                     |
| ---------------------- | ----------------------------------------------------------- |
| `field.update`         | Realtime value change in a regular block.                   |
| `field.validate`       | Request backend validation for a field.                     |
| `form.apply`           | Submit all values from a form block.                        |
| `button.press`         | A declared button was pressed.                              |
| `autocomplete.request` | Request completions for an autocomplete-enabled text field. |

Every message belongs to one menu instance via `menuId`.
Generations are backend-assigned numbers that let frontends ignore stale state:

- Increment `menuGeneration` when the set of blocks changes.
- Increment a block `generation` when the block structure or item configuration changes.
- Keep block IDs and field IDs stable so frontend state can survive ordinary refreshes.

## Create a basic menu
A menu contains blocks.
A block contains ordered items.
The smallest useful menu has one block with one field:

```json
{
  "type": "menu.snapshot",
  "menuId": "settings",
  "menuGeneration": 1,
  "force": true,
  "blocks": [
    {
      "id": "main",
      "order": 10,
      "generation": 1,
      "form": false,
      "items": [
        {
          "type": "field",
          "id": "name",
          "label": "Name",
          "kind": "text",
          "value": "Ada"
        }
      ]
    }
  ]
}
```

The equivalent Go shape is:

```go
send(formular.MenuSnapshotMessage{
    MessageBase: formular.MessageBase{
        Type:           formular.MessageMenuSnapshot,
        MenuID:         "settings",
        MenuGeneration: 1,
    },
    Force: true,
    Blocks: []formular.Block{
        {
            ID:         "main",
            Order:      10,
            Generation: 1,
            Form:       false,
            Items: []formular.Item{
                {
                    Type:  formular.ItemField,
                    ID:    "name",
                    Label: "Name",
                    Field: &formular.Field{
                        Kind:  formular.FieldText,
                        Value: "Ada",
                    },
                },
            },
        },
    },
})
```

For ordinary backend code, the Go package also provides constructors and field
builders that produce the same wire structs without repeating the envelope and
field-item boilerplate:

```go
send(formular.ForcedMenuSnapshot("settings", 1, formular.Block{
    ID:         "main",
    Order:      10,
    Generation: 1,
    Items: []formular.Item{
        formular.TextField(
            "name",
            "Name",
            "Ada",
            formular.Required,
            formular.Placeholder("Display name"),
        ),
    },
}))
```

In JSON, field properties are flattened into the field item: `kind`, `value`,
`readonly`, `required`, and so on live next to `type`, `id`, and `label`.

## Split menus into multiple blocks
Blocks are independently replaceable, so prefer several smaller blocks over one
large block when parts of the menu change at different rates.

```json
{
  "type": "menu.snapshot",
  "menuId": "node-42",
  "menuGeneration": 1,
  "blocks": [
    {
      "id": "connection",
      "order": 10,
      "generation": 1,
      "form": true,
      "collapsible": true,
      "items": [
        { "type": "header", "id": "title", "text": "Connection" },
        { "type": "field", "id": "host", "label": "Host", "kind": "text", "value": "127.0.0.1" },
        { "type": "field", "id": "port", "label": "Port", "kind": "int", "value": 8080 }
      ]
    },
    {
      "id": "status",
      "order": 20,
      "generation": 1,
      "form": false,
      "items": [
        { "type": "label", "id": "state", "text": "ready", "format": "plain" }
      ]
    }
  ]
}
```

To refresh only the status block, send a `block.snapshot`:

```json
{
  "type": "block.snapshot",
  "menuId": "node-42",
  "menuGeneration": 1,
  "blockGeneration": 2,
  "block": {
    "id": "status",
    "order": 20,
    "generation": 2,
    "form": false,
    "items": [
      { "type": "label", "id": "state", "text": "listening on 127.0.0.1:8080" }
    ]
  }
}
```

To remove a block, send:

```json
{
  "type": "block.delete",
  "menuId": "node-42",
  "menuGeneration": 2,
  "blockId": "status"
}
```

## Add help text
Any item may include `help`. Keep it plain text.
Frontends may render it as a tooltip, popover, side panel, or another
native hint UI.

```json
{
  "type": "field",
  "id": "url",
  "label": "URL",
  "help": "The absolute URL that will be requested when Send is pressed.",
  "kind": "text",
  "placeholder": "https://example.com/"
}
```

Headers, labels, buttons, progress bars, logs, fields, and array element items
can all use `help`.

## Use basic field types
### Text
Text fields use string values.

```json
{
  "type": "field",
  "id": "url",
  "label": "URL",
  "kind": "text",
  "value": "https://example.com/",
  "placeholder": "https://example.com/",
  "required": true
}
```

Useful text options:

```json
{
  "kind": "text",
  "secret": true,
  "multiline": true,
  "subtype": "email",
  "autocomplete": { "enabled": true, "tag": "email" }
}
```

`secret` asks for password-style display. `multiline` allows newlines.
`subtype` refines the input, for example `email` or `filepath`.

### Int
Int fields use JSON integer values.

```json
{
  "type": "field",
  "id": "port",
  "label": "Port",
  "kind": "int",
  "value": 8080,
  "min": 1,
  "max": 65535
}
```

A backend should still validate received values.
Frontend constraints are user experience, not trust boundaries.

### Checkbox
Checkbox fields use boolean values.

```json
{
  "type": "field",
  "id": "enabled",
  "label": "Enabled",
  "kind": "checkbox",
  "value": true
}
```

When the user changes this field in a regular block, expect:

```json
{
  "type": "field.update",
  "menuId": "node-42",
  "menuGeneration": 1,
  "blockGeneration": 1,
  "field": { "blockId": "main", "fieldId": "enabled" },
  "value": false
}
```

### Radio
Radio fields use one selected value from `allowedValues`.

```json
{
  "type": "field",
  "id": "method",
  "label": "Method",
  "kind": "radio",
  "value": "GET",
  "allowedValues": ["GET", "POST", "PUT", "DELETE"]
}
```

Values do not have to be strings, but strings are easiest to interoperate with.

### Range
Range fields use numeric values.
Frontends may render them as sliders, steppers, numeric inputs, or another
native range control.

```json
{
  "type": "field",
  "id": "volume",
  "label": "Volume",
  "kind": "range",
  "value": 42,
  "min": 0,
  "max": 100
}
```

## Handle regular block realtime input
A block with `form: false` is a regular block.
Frontends send `field.update` messages as the user edits fields.
They may debounce, so do not rely on receiving every intermediate keystroke.

Example regular block:

```json
{
  "id": "request",
  "order": 10,
  "generation": 1,
  "form": false,
  "items": [
    { "type": "field", "id": "url", "label": "URL", "kind": "text", "value": "http://127.0.0.1:8080/" },
    { "type": "field", "id": "method", "label": "Method", "kind": "radio", "value": "GET", "allowedValues": ["GET", "POST"] },
    { "type": "field", "id": "body", "label": "Body", "kind": "text", "multiline": true },
    { "type": "button", "id": "send", "label": "Send" }
  ]
}
```

Frontend message:

```json
{
  "type": "field.update",
  "menuId": "node-42",
  "menuGeneration": 1,
  "blockGeneration": 1,
  "field": { "blockId": "request", "fieldId": "url" },
  "value": "https://example.com/"
}
```

Typical backend handling:

```go
func onMessage(message any) error {
    msg, ok := message.(formular.FieldUpdateMessage)
    if !ok || msg.MenuID != menuID {
        return nil
    }
    if msg.Field.BlockID != "request" {
        return nil
    }

    switch msg.Field.FieldID {
    case "url":
        state.URL, _ = msg.Value.(string)
    case "method":
        state.Method, _ = msg.Value.(string)
    case "body":
        state.Body, _ = msg.Value.(string)
    }

    // Optionally refresh labels, statuses, or dependent fields.
    sendRequestBlock()
    return nil
}
```

Use regular blocks for live controls, node settings that should apply immediately, and fields that drive other UI state.

## Handle form submit input
A block with `form: true` is applied as a whole.
The frontend owns Apply and Reset controls for the block.
Do not add your own normal `Apply` button unless it has separate application meaning.

Example form block:

```json
{
  "id": "listen",
  "order": 10,
  "generation": 1,
  "form": true,
  "items": [
    { "type": "field", "id": "host", "label": "Host", "kind": "text", "value": "127.0.0.1" },
    { "type": "field", "id": "port", "label": "Port", "kind": "int", "value": 8080 }
  ]
}
```

Frontend message:

```json
{
  "type": "form.apply",
  "menuId": "node-42",
  "menuGeneration": 1,
  "blockGeneration": 1,
  "blockId": "listen",
  "values": {
    "host": "0.0.0.0",
    "port": 9000
  }
}
```

Typical backend handling:

```go
func onMessage(message any) error {
    msg, ok := message.(formular.FormApplyMessage)
    if !ok || msg.MenuID != menuID || msg.BlockID != "listen" {
        return nil
    }

    host, _ := msg.Values["host"].(string)
    port, ok := intValue(msg.Values["port"])
    if !ok || port < 1 || port > 65535 {
        sendFieldStatus("listen", "port", formular.StatusError, "Port must be between 1 and 65535")
        return nil
    }

    state.Host = host
    state.Port = port
    restartServer()
    sendListenBlock()
    return nil
}
```

Use form blocks when the user should edit several values and apply them atomically.

## Add buttons
Buttons are ordinary items.
A button sends `button.press` with its owning block and button ID.

```json
{
  "type": "button",
  "id": "send",
  "label": "Send request",
  "help": "Send the request using the current URL, method, and body."
}
```

Frontend message:

```json
{
  "type": "button.press",
  "menuId": "node-42",
  "menuGeneration": 1,
  "blockGeneration": 1,
  "blockId": "request",
  "buttonId": "send"
}
```

Backend handling:

```go
msg, ok := message.(formular.ButtonPressMessage)
if ok && msg.MenuID == menuID && msg.BlockID == "request" && msg.ButtonID == "send" {
    sendHTTPRequestFromCurrentState()
}
```

Set `inactive: true` on a button to render it disabled:

```json
{ "type": "button", "id": "send", "label": "Send", "inactive": true }
```

## Use file inputs
File fields upload file content as base64 text in the JSON value.
They may define `maxBytes` and accepted MIME types.

```json
{
  "type": "field",
  "id": "avatar",
  "label": "Avatar file",
  "kind": "file",
  "maxBytes": 4098,
  "accept": ["image/png", "image/jpeg"],
  "help": "Small PNG or JPEG image."
}
```

A form submit containing a selected file looks like:

```json
{
  "type": "form.apply",
  "menuId": "profile",
  "blockId": "profile",
  "values": {
    "avatar": "iVBORw0KGgoAAAANSUhEUgAA..."
  }
}
```

Backend rules:
- Treat the base64 string as untrusted input.
- Decode it backend-side and enforce the real byte limit again.
- Verify the content type if it matters.
- Do not rely on realtime validation for file fields; validate them on submit or on explicit backend action.

## Validate fields and send field status
Set `validate: true` on fields that require backend validation while the user
edits them.

```json
{
  "type": "field",
  "id": "email",
  "label": "Email",
  "kind": "text",
  "value": "admin@example.com",
  "subtype": "email",
  "required": true,
  "validate": true
}
```

The frontend may send:

```json
{
  "type": "field.validate",
  "menuId": "profile",
  "menuGeneration": 1,
  "blockGeneration": 1,
  "field": { "blockId": "profile", "fieldId": "email" },
  "value": "ada@example.com"
}
```

Answer with `field.status`:

```json
{
  "type": "field.status",
  "menuId": "profile",
  "menuGeneration": 1,
  "blockGeneration": 1,
  "field": { "blockId": "profile", "fieldId": "email" },
  "status": "ok",
  "statusText": "Looks good"
}
```

Possible statuses are:

| Status  | Meaning                                         |
| ------- | ----------------------------------------------- |
| `unset` | No backend opinion yet.                         |
| `ok`    | Value is accepted.                              |
| `warn`  | Value is accepted but suspicious or incomplete. |
| `error` | Value is invalid.                               |

You can also include status in the original field snapshot:

```json
{
  "type": "field",
  "id": "url",
  "label": "URL",
  "kind": "text",
  "value": "http://localhost/",
  "status": "warn",
  "statusText": "Localhost will only work on the same machine."
}
```

`field.status` can update readonly state too:

```json
{
  "type": "field.status",
  "menuId": "node-42",
  "menuGeneration": 1,
  "blockGeneration": 1,
  "field": { "blockId": "listen", "fieldId": "port" },
  "status": "unset",
  "readonly": true
}
```

Still validate everything again when processing `field.update` or `form.apply`.
Frontend validation only improves UI feedback.

## Use readonly and inactive state
Use field-level `readonly` when a value is visible but should not be edited:

```json
{
  "type": "field",
  "id": "host",
  "label": "Host",
  "kind": "text",
  "value": "127.0.0.1",
  "readonly": true
}
```

Use block-level `inactive` to disable all interaction in a block:

```json
{
  "id": "danger-zone",
  "order": 90,
  "generation": 1,
  "inactive": true,
  "items": [
    { "type": "button", "id": "delete", "label": "Delete" }
  ]
}
```

Readonly is useful when a backend value is controlled by another source.
For example, a node menu can allow editing `host` and `port` only when no link
is attached, and make those fields readonly while a linked value is driving them.

## Add array fields
Array fields represent ordered collections of templated elements.
They are useful for repeatable configuration: routes, headers, template parts,
servers, environment variables, and so on.

An array field has:
- `templates`: allowed element shapes.
- `elements`: current rendered element snapshots.
- A value shape sent back as an array of `{ id, template, values }` objects.

Example: a format string made from text parts and value placeholders.

```json
{
  "type": "field",
  "id": "parts",
  "label": "Template",
  "kind": "array",
  "templates": [
    {
      "name": "text",
      "label": "Text",
      "items": [
        { "type": "field", "id": "text", "label": "Text", "kind": "text", "multiline": true }
      ]
    },
    {
      "name": "value",
      "label": "Value",
      "items": [
        { "type": "field", "id": "name", "label": "Name", "kind": "text", "value": "Value" },
        { "type": "field", "id": "type", "label": "Type", "kind": "radio", "value": "pasta/string", "allowedValues": ["pasta/string", "pasta/int", "pasta/float", "pasta/bool"] }
      ]
    }
  ],
  "elements": [
    {
      "id": "text-1",
      "template": "text",
      "items": [
        { "type": "field", "id": "text", "label": "Text", "kind": "text", "value": "Hello, " }
      ]
    },
    {
      "id": "value-1",
      "template": "value",
      "items": [
        { "type": "field", "id": "name", "label": "Name", "kind": "text", "value": "user" },
        { "type": "field", "id": "type", "label": "Type", "kind": "radio", "value": "pasta/string", "allowedValues": ["pasta/string", "pasta/int", "pasta/float", "pasta/bool"] }
      ]
    }
  ]
}
```

The corresponding value shape is:

```json
[
  {
    "id": "text-1",
    "template": "text",
    "values": { "text": "Hello, " }
  },
  {
    "id": "value-1",
    "template": "value",
    "values": { "name": "user", "type": "pasta/string" }
  }
]
```

### Address fields inside array elements
A nested field is addressed by the final `fieldId` plus an `elementPath`.

```json
{
  "type": "field.update",
  "menuId": "node-42",
  "field": {
    "blockId": "template",
    "fieldId": "type",
    "elementPath": [
      { "arrayFieldId": "parts", "elementId": "value-1" }
    ]
  },
  "value": "pasta/int"
}
```

Backend handling pattern:

```go
msg, ok := message.(formular.FieldUpdateMessage)
if !ok || msg.Field.BlockID != "template" {
    return nil
}

// Top-level array value changed: add/remove/reorder elements or edit many fields at once.
if msg.Field.FieldID == "parts" && len(msg.Field.ElementPath) == 0 {
    parts, ok := formular.ArrayElementValuesFromAny(msg.Value)
    if ok {
        state.Parts = parts
        sendTemplateBlock()
    }
    return nil
}

// One field inside one array element changed.
if len(msg.Field.ElementPath) > 0 {
    segment := msg.Field.ElementPath[len(msg.Field.ElementPath)-1]
    if segment.ArrayFieldID == "parts" {
        updatePartField(segment.ElementID, msg.Field.FieldID, msg.Value)
        sendTemplateBlock()
    }
}
```

Array elements cannot be form blocks by themselves.
The containing block decides whether input is realtime or submitted as one form.

### Address buttons inside array elements
Buttons inside elements send the same `elementPath` on `button.press`:

```json
{
  "type": "button.press",
  "menuId": "node-42",
  "blockId": "servers",
  "elementPath": [
    { "arrayFieldId": "servers", "elementId": "server-1" }
  ],
  "buttonId": "generate"
}
```

Backend handling:
```go
msg, ok := message.(formular.ButtonPressMessage)
if !ok || msg.BlockID != "servers" || msg.ButtonID != "generate" {
    return nil
}
if len(msg.ElementPath) == 0 {
    return nil
}
segment := msg.ElementPath[len(msg.ElementPath)-1]
if segment.ArrayFieldID == "servers" {
    generateServerValues(segment.ElementID)
    sendServersBlock()
}
```

### Update labels inside array elements
There is no separate “label update” message for labels inside array elements.
Labels, buttons, and non-field element items are part of the block snapshot.
To change them, rebuild the affected array element and send a new
`block.snapshot` for the owning block.

### Process array input in regular blocks
For regular blocks, frontends may send either a top-level array `field.update`
or nested `field.update` messages for fields inside elements.
A robust backend should handle both:

```go
switch {
case msg.Field.FieldID == "servers" && len(msg.Field.ElementPath) == 0:
    // Whole array changed: add, remove, reorder, or bulk edit.
    servers, ok := formular.ArrayElementValuesFromAny(msg.Value)
    if ok {
        state.Servers = servers
    }

case len(msg.Field.ElementPath) > 0:
    // One field inside one element changed.
    segment := msg.Field.ElementPath[len(msg.Field.ElementPath)-1]
    if segment.ArrayFieldID == "servers" {
        setServerField(segment.ElementID, msg.Field.FieldID, msg.Value)
    }
}
```

### Process array input in form blocks
For form blocks, the array value is included in `form.apply.values`:

```json
{
  "type": "form.apply",
  "menuId": "node-42",
  "blockId": "routing",
  "values": {
    "servers": [
      { "id": "server-1", "template": "http", "values": { "host": "localhost", "port": 8080 } },
      { "id": "server-2", "template": "database", "values": { "driver": "postgres", "dsn": "postgres://localhost/app" } }
    ]
  }
}
```

Validate every element:
- `id` must exist and be unique inside the array.
- `template` must be one of the declared templates.
- `values` must contain only fields that are valid for that template.
- Each nested value must have the expected type and range.

The Go helpers use the same scalar decoding rules for top-level and array
values:

```go
servers, ok := formular.ArrayElementValuesFromAny(msg.Values["servers"])
if !ok {
    return fmt.Errorf("invalid server array")
}
for _, server := range servers {
    host, _ := formular.ArrayElementStringValue(server, "host")
    port, _ := formular.ArrayElementIntValue(server, "port")
    _ = host
    _ = port
}
```

Use `StringValue`, `IntValue`, `FloatValue`, and `BoolValue` for non-array form
values that may come from JSON-decoded payloads.

## Add autocomplete
Autocomplete is enabled per text field.

```json
{
  "type": "field",
  "id": "timezone",
  "label": "Timezone",
  "kind": "text",
  "value": "UTC",
  "autocomplete": { "enabled": true, "tag": "timezone" }
}
```

The frontend may send:

```json
{
  "type": "autocomplete.request",
  "menuId": "profile",
  "field": { "blockId": "profile", "fieldId": "timezone" },
  "prefix": "Europe/"
}
```

The backend replies with complete candidate values, not suffixes:

```json
{
  "type": "autocomplete.hints",
  "menuId": "profile",
  "menuGeneration": 1,
  "blockGeneration": 1,
  "field": { "blockId": "profile", "fieldId": "timezone" },
  "prefix": "Europe/",
  "hints": ["Europe/Tbilisi", "Europe/Berlin"]
}
```

Backend pattern:

```go
msg, ok := message.(formular.AutocompleteRequestMessage)
if !ok || msg.MenuID != menuID {
    return nil
}

hints := complete(msg.Field.FieldID, msg.Prefix)
send(formular.AutocompleteHintsMessage{
    MessageBase: formular.MessageBase{
        Type:            formular.MessageAutocompleteHints,
        MenuID:          msg.MenuID,
        MenuGeneration:  currentMenuGeneration,
        BlockGeneration: currentBlockGeneration(msg.Field.BlockID),
    },
    Field:  msg.Field,
    Prefix: msg.Prefix,
    Hints:  hints,
})
```

## Add copyable blocks, fields, and elements
`copyable` declares text for an explicit frontend copy action.
It is not the same as ordinary text selection copy.

Block-level copy:

```json
{
  "id": "request",
  "order": 10,
  "generation": 1,
  "copyable": { "text": "curl https://example.com/" },
  "items": []
}
```

Field-level copy:

```json
{
  "type": "field",
  "id": "token",
  "label": "Token",
  "kind": "text",
  "readonly": true,
  "value": "abc123",
  "copyable": { "text": "abc123" }
}
```

Array element copy:

```json
{
  "id": "server-1",
  "template": "http",
  "copyable": { "text": "localhost:8080" },
  "items": []
}
```

Keep copy text synchronized when you send a fresh block snapshot.

## Display labels, progress bars, and logs
### Headers
Headers are simple section headings. They have no levels.

```json
{ "type": "header", "id": "title", "text": "Profile form" }
```

### Text labels
Labels display readonly text. Use `format` to choose presentation:

```json
{ "type": "label", "id": "plain", "text": "Plain text", "format": "plain" }
```

```json
{ "type": "label", "id": "markdown", "text": "Markdown **label** with `code`.", "format": "markdown" }
```

```json
{ "type": "label", "id": "code", "text": "go test ./...", "format": "code", "syntax": "sh" }
```

Use labels for explanations, current status, rendered output, and read-only summaries. To update a label, send a new `block.snapshot` for the block that owns it.

### Progress bars
Progress bars are display-only items. `progress` is a percentage from 0 to 100.

```json
{
  "type": "progressbar",
  "id": "sync-progress",
  "label": "Background sync",
  "progress": 70
}
```

To update progress, send a new `block.snapshot` with the changed progress value.

### Logs
Logs display readonly structured lines. Supported levels are 
`trace`, `debug`, `info`, `warn`, `error`, and `panic`.

```json
{
  "type": "logs",
  "id": "worker",
  "label": "Worker",
  "logs": [
    { "level": "info", "text": "worker restarting" },
    { "level": "warn", "text": "waiting for network" }
  ]
}
```

For long-running logs, keep a bounded tail in backend state and resend the logs 
block when new lines arrive.

## Build backend receive logic around message type
A backend usually dispatches by message type first, then by `menuId`, `blockId`,
field reference, and optional array `elementPath`.

```go
func receive(message any) error {
    switch msg := message.(type) {
    case formular.FieldUpdateMessage:
        return handleFieldUpdate(msg)
    case formular.FieldValidateMessage:
        return handleFieldValidate(msg)
    case formular.FormApplyMessage:
        return handleFormApply(msg)
    case formular.ButtonPressMessage:
        return handleButtonPress(msg)
    case formular.AutocompleteRequestMessage:
        return handleAutocomplete(msg)
    default:
        return nil
    }
}
```

Always reject or ignore unexpected messages:

```go
func handleFieldUpdate(msg formular.FieldUpdateMessage) error {
    if msg.MenuID != menuID {
        return nil
    }
    if stale(msg.MenuGeneration, msg.BlockGeneration) {
        return nil
    }
    if !knownBlock(msg.Field.BlockID) || !knownField(msg.Field) {
        return nil
    }
    return applyFieldValue(msg.Field, msg.Value)
}
```

Do not assume the frontend sent valid data.
Check value types, field IDs, array element IDs, templates, and ranges every time.

## Common backend patterns

### Editable node constant
A constant-like object can use one regular block with one field.
On `field.update`, store the value, refresh the block, and broadcast the new
value to the rest of your application.

```json
{
  "id": "state",
  "order": 10,
  "generation": 1,
  "form": false,
  "items": [
    { "type": "field", "id": "value", "label": "Value", "kind": "text", "value": "hello" }
  ]
}
```

### Form-controlled service
A service-like object can use a form block for configuration and a separate
logs block for runtime state:

```json
{
  "blocks": [
    {
      "id": "listen",
      "order": 10,
      "generation": 1,
      "form": true,
      "items": [
        { "type": "field", "id": "host", "label": "Host", "kind": "text", "value": "127.0.0.1" },
        { "type": "field", "id": "port", "label": "Port", "kind": "int", "value": 8080 }
      ]
    },
    {
      "id": "logs",
      "order": 20,
      "generation": 1,
      "form": false,
      "items": [
        { "type": "logs", "id": "worker", "label": "Worker", "logs": [] }
      ]
    }
  ]
}
```

When runtime state changes, resend only the logs block. When configuration is applied, validate it, update backend state, restart the service, and resend the listen block.

### Linked or externally-controlled field
When a value can come either from menu input or from another backend source,
keep the same field visible in both cases:

- no external source: field is editable;
- external source attached: field is readonly and displays the external value;
- external value changes: resend the block snapshot or send `field.status` with `readonly` if only readonly state changed.

```json
{
  "type": "field",
  "id": "url",
  "label": "URL",
  "kind": "text",
  "value": "https://example.com/",
  "readonly": true,
  "help": "URL is controlled by an attached string input."
}
```

## Checklist
Before sending a menu:

- Use stable IDs for menus, blocks, fields, buttons, templates, and array elements.
- Set `form: true` only when the whole block should be applied together.
- Set `form: false` for realtime controls.
- Include `help` where a frontend hint would prevent confusion.
- Keep blocks small enough to refresh independently.
- Include `readonly`, `inactive`, `status`, and `statusText` directly in snapshots when they are already known.

Before processing frontend input:

- Check `menuId`.
- Check generation freshness if your application can have stale frontends.
- Check `blockId`, `fieldId`, `buttonId`, and `elementPath` against known state.
- Parse and validate every value backend-side.
- For arrays, validate element IDs, templates, and nested values.
- Prefer ignoring malformed or irrelevant messages over crashing.

When updating frontend state:

- Send `field.status` for validation, status text, and readonly changes.
- Send `block.snapshot` when item text, labels, progress, logs, values, array elements, buttons, or structure changed.
- Send `menu.snapshot` when the block set changed or frontend state must be reset.
- Use `force: true` when the frontend should discard remembered local state such as collapse state.
