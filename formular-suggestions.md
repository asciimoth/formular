# Formular API Suggestions

This document collects suggestions for reducing repeated consumer-side
boilerplate when building dynamic forms and handling form messages with
Formular. The common theme is that applications often repeat the same low-level
construction and decoding logic around Formular's core types. Moving a few of
those patterns into Formular would make consumers shorter, more consistent, and
less likely to diverge on edge cases.

## Goals

- Provide canonical constructors for common backend messages.
- Provide field and item builders for common controls.
- Provide typed value decoding helpers for values received from form events.
- Provide canonical decoding and instantiation helpers for array fields.
- Keep host-application concepts out of Formular; callers should pass plain
  menu IDs, block IDs, field references, and values.

## Message Constructors

Consumers repeatedly assemble the same message envelopes by hand: menu
snapshots, forced menu snapshots, block snapshots, field status messages, and
autocomplete responses. These are mechanical constructions, but handwritten
versions can drift in default generation handling, status encoding, and field
reference shape.

Suggested API shape:

```go
formular.MenuSnapshot(menuID string, generation uint64, blocks ...Block) MenuSnapshotMessage
formular.ForcedMenuSnapshot(menuID string, generation uint64, blocks ...Block) MenuSnapshotMessage
formular.BlockSnapshot(menuID string, menuGen, blockGen uint64, block Block) BlockSnapshotMessage
formular.FieldStatusFromError(menuID string, menuGen, blockGen uint64, ref FieldRef, err error) FieldStatusMessage
formular.AutocompleteHints(menuID string, menuGen uint64, ref FieldRef, prefix string, hints []string) AutocompleteHintsMessage
```

Expected behavior:

- Constructors should set the correct message type and envelope fields.
- `FieldStatusFromError` should produce an OK status for `nil` and an error
  status with useful text for non-nil errors.
- `ForcedMenuSnapshot` should be the canonical way to request a full client
  refresh when the existing protocol supports that distinction.
- Callers remain responsible for choosing menu and block IDs.

## Field And Item Builders

Many consumers build common fields directly from struct literals. This makes
forms verbose and encourages small differences in how options such as readonly,
secret, validation status, placeholders, allowed values, and multiline behavior
are represented.

Suggested API shape:

```go
formular.TextField(id, label, value string, opts ...FieldOption) Item
formular.IntField(id, label string, value int, opts ...FieldOption) Item
formular.FloatField(id, label string, value float64, opts ...FieldOption) Item
formular.CheckboxField(id, label string, value bool, opts ...FieldOption) Item
formular.RadioField(id, label string, value string, options []Choice, opts ...FieldOption) Item
formular.ReadonlyTextField(id, label, value string, opts ...FieldOption) Item
formular.SecretTextField(id, label, value string, opts ...FieldOption) Item
formular.MultilineTextField(id, label, value string, opts ...FieldOption) Item
formular.Button(id, label string, opts ...ItemOption) Item
formular.PlainLabel(id, text string) Item
formular.Logs(id, label string, lines []LogLine) Item
```

Useful options:

- `Readonly`
- `Required`
- `Secret`
- `Multiline`
- `Placeholder`
- `Help`
- `Validation`
- `Status`
- `StatusText`
- `AllowedValues`
- `Autocomplete`

Design notes:

- Builders should remain thin wrappers over the existing Formular model, not a
  separate form DSL.
- Options should compose predictably and should not hide the underlying field
  identity or type.
- The returned value should be the normal `Item` type so consumers can freely
  mix builder-created items with manually constructed items.

## Typed Form Value Decoding

Form event payloads often arrive as `any`, especially after JSON-like transport
or storage layers. Consumers currently need small conversion helpers for common
scalar types and must decide individually which numeric and string conversions
are acceptable.

Suggested API shape:

```go
formular.StringValue(v any) (string, bool)
formular.IntValue(v any) (int, bool)
formular.FloatValue(v any) (float64, bool)
formular.BoolValue(v any) (bool, bool)
```

Suggested semantics:

- String decoding accepts strings and may optionally accept values implementing
  `fmt.Stringer` only if that is consistent with the rest of the library.
- Integer decoding accepts signed and unsigned integer types when the value fits
  in `int`; it may accept whole-number `float64` values produced by JSON
  decoding.
- Float decoding accepts Go integer and floating-point numeric types.
- Boolean decoding accepts booleans and should avoid broad string parsing unless
  Formular already treats string booleans as valid field values.
- Failed conversion returns the zero value and `false`.

Having these semantics in Formular would let consumers handle payloads the same
way across apply, validate, update, and array-element code paths.

## Array Field Value Decoding

Array fields tend to create the most duplication because event payloads can
contain nested element values, field IDs, and per-element maps. A canonical
decoder would remove a common source of fragile type assertions.

Suggested API shape:

```go
formular.ArrayElementValueFromAny(v any) (ArrayElementValue, bool)
formular.ArrayElementValuesFromAny(v any) ([]ArrayElementValue, bool)
formular.ArrayElementFieldValue(v any, fieldID string) (any, bool)
formular.ArrayElementStringValue(v any, fieldID string) (string, bool)
formular.ArrayElementIntValue(v any, fieldID string) (int, bool)
formular.ArrayElementBoolValue(v any, fieldID string) (bool, bool)
```

Expected behavior:

- Accept the canonical in-memory array element value type.
- Accept map-shaped values produced by JSON-like decoding when field names and
  value shapes are unambiguous.
- Preserve element identity where the protocol carries one.
- Return `false` rather than panicking on malformed input.
- Use the same scalar decoding rules as the top-level typed value helpers.

This is likely the highest-value area because every consumer using editable
arrays otherwise needs to reverse-engineer the same payload shapes.

## Array Template Instantiation

Consumers with editable arrays often define the same field list twice: once as
the empty template and once for each existing element with values overlaid. A
small helper could instantiate a template with per-field values while preserving
the same field metadata.

Suggested API shape:

```go
formular.InstantiateArrayTemplate(template ArrayTemplate, id string, values map[string]any) ArrayElement
```

Expected behavior:

- Copy the template field layout.
- Apply values by field ID.
- Leave unspecified fields at their template defaults.
- Preserve validation, labels, options, readonly flags, and other metadata.
- Avoid mutating the original template.

This keeps array form definitions single-sourced without requiring consumers to
build their own mini templating layer.

## Form Dispatch Helpers

Consumers frequently route form messages through the same guard sequence:
matching menu ID, matching block ID, top-level field checks, typed value
extraction, and then a field-specific handler. Formular could provide a generic
message dispatch helper as long as it stays independent of any host
application's node, routing, or ownership model.

Possible API direction:

```go
type FormHandler struct {
    MenuID  string
    BlockID string

    OnApply    func(fieldID string, value any) error
    OnValidate func(fieldID string, value any) error
    OnUpdate   func(fieldID string, value any) error
}

func DispatchFormMessage(msg Message, h FormHandler) (handled bool, err error)
```

Design notes:

- The helper should be optional. Consumers with complex routing should still be
  able to inspect messages manually.
- The helper should not assume where menu IDs come from or how updates are sent
  back to clients.
- Typed accessors should be usable inside handlers rather than being coupled
  directly to dispatch.

## Compatibility Considerations

These additions can be introduced as convenience APIs around existing public
types. They do not need to replace direct struct construction or manual message
handling.

Recommended rollout:

1. Add scalar and array value decoders first, because they define canonical
   payload interpretation and are useful without changing form construction.
2. Add message constructors next, keeping them as thin wrappers around existing
   message structs.
3. Add field and item builders with functional options.
4. Add array template instantiation once the canonical array value semantics are
   settled.
5. Consider dispatch helpers last, because routing needs vary more across
   applications.

## Summary

The most useful upstream additions are small, canonical helpers rather than a
new abstraction layer. Formular already owns the protocol shape, field model,
and array value conventions, so it is the right place to centralize message
construction, common item builders, typed form value decoding, and array element
handling.
