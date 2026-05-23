// Package formular defines wire types for the Formular JSON menu and form
// protocol.
package formular

import "reflect"

// Message type constants used in Formular JSON envelopes.
const (
	MessageMenuSnapshot        = "menu.snapshot"
	MessageBlockSnapshot       = "block.snapshot"
	MessageBlockDelete         = "block.delete"
	MessageFieldStatus         = "field.status"
	MessageAutocompleteHints   = "autocomplete.hints"
	MessageFieldUpdate         = "field.update"
	MessageFieldValidate       = "field.validate"
	MessageFormApply           = "form.apply"
	MessageButtonPress         = "button.press"
	MessageAutocompleteRequest = "autocomplete.request"
)

// Item type constants used by menu and array element content.
const (
	ItemHeader      = "header"
	ItemLabel       = "label"
	ItemProgressbar = "progressbar"
	ItemLogs        = "logs"
	ItemButton      = "button"
	ItemField       = "field"
)

// Field kind constants supported by the base DSL.
const (
	FieldText     = "text"
	FieldInt      = "int"
	FieldFloat    = "float"
	FieldFile     = "file"
	FieldCheckbox = "checkbox"
	FieldRadio    = "radio"
	FieldRange    = "range"
	FieldArray    = "array"
)

// Text format constants for label rendering.
const (
	TextPlain    = "plain"
	TextMarkdown = "markdown"
	TextCode     = "code"
)

// Log level constants for logs item rendering.
const (
	LogTrace = "trace"
	LogDebug = "debug"
	LogInfo  = "info"
	LogWarn  = "warn"
	LogError = "error"
	LogPanic = "panic"
)

// Validation status constants set by a backend.
const (
	StatusUnset = "unset"
	StatusOK    = "ok"
	StatusWarn  = "warn"
	StatusError = "error"
)

// MessageBase contains fields common to all Formular protocol messages.
type MessageBase struct {
	// Type identifies the concrete message shape.
	Type string `json:"type"`
	// MenuID identifies the menu instance the message belongs to.
	MenuID string `json:"menuId"`
	// MenuGeneration is the backend-assigned menu structure generation.
	MenuGeneration uint64 `json:"menuGeneration,omitempty"`
	// BlockGeneration is the backend-assigned block structure generation.
	BlockGeneration uint64 `json:"blockGeneration,omitempty"`
}

// Copy returns a deep copy of m.
func (m MessageBase) Copy() MessageBase {
	return m
}

// MenuSnapshotMessage is sent by a backend to initialize or replace a menu.
type MenuSnapshotMessage struct {
	MessageBase
	// Force tells the frontend to discard remembered state and reinitialize.
	Force bool `json:"force,omitempty"`
	// Blocks is the complete block list for this menu snapshot.
	Blocks []Block `json:"blocks"`
}

// Copy returns a deep copy of m.
func (m MenuSnapshotMessage) Copy() MenuSnapshotMessage {
	m.MessageBase = m.MessageBase.Copy()
	m.Blocks = copyBlocks(m.Blocks)
	return m
}

// BlockSnapshotMessage is sent by a backend to create or replace one block.
type BlockSnapshotMessage struct {
	MessageBase
	// Block is the full current snapshot of the addressed block.
	Block Block `json:"block"`
}

// Copy returns a deep copy of m.
func (m BlockSnapshotMessage) Copy() BlockSnapshotMessage {
	m.MessageBase = m.MessageBase.Copy()
	m.Block = m.Block.Copy()
	return m
}

// BlockDeleteMessage is sent by a backend to remove one block.
type BlockDeleteMessage struct {
	MessageBase
	// BlockID identifies the block to remove.
	BlockID string `json:"blockId"`
}

// Copy returns a deep copy of m.
func (m BlockDeleteMessage) Copy() BlockDeleteMessage {
	m.MessageBase = m.MessageBase.Copy()
	return m
}

// FieldStatusMessage is sent by a backend to update a field status or readonly flag.
type FieldStatusMessage struct {
	MessageBase
	// Field identifies the top-level or nested field being updated.
	Field FieldRef `json:"field"`
	// Status is the backend validation state to display.
	Status string `json:"status"`
	// StatusText is optional explanatory text for the status.
	StatusText string `json:"statusText,omitempty"`
	// Readonly optionally changes whether the field accepts user input.
	Readonly *bool `json:"readonly,omitempty"`
}

// Copy returns a deep copy of m.
func (m FieldStatusMessage) Copy() FieldStatusMessage {
	m.MessageBase = m.MessageBase.Copy()
	m.Field = m.Field.Copy()
	m.Readonly = copyPtr(m.Readonly)
	return m
}

// AutocompleteHintsMessage is sent by a backend with possible completions.
type AutocompleteHintsMessage struct {
	MessageBase
	// Field identifies the text field the hints belong to.
	Field FieldRef `json:"field"`
	// Prefix is the input prefix the backend used to compute hints.
	Prefix string `json:"prefix"`
	// Hints contains complete candidate values, not suffixes.
	Hints []string `json:"hints"`
}

// Copy returns a deep copy of m.
func (m AutocompleteHintsMessage) Copy() AutocompleteHintsMessage {
	m.MessageBase = m.MessageBase.Copy()
	m.Field = m.Field.Copy()
	m.Hints = copySlice(m.Hints)
	return m
}

// FieldUpdateMessage is sent by a frontend for realtime field changes.
type FieldUpdateMessage struct {
	MessageBase
	// Field identifies the field whose value changed.
	Field FieldRef `json:"field"`
	// Value is the current field value after frontend normalization.
	Value any `json:"value"`
}

// Copy returns a deep copy of m.
func (m FieldUpdateMessage) Copy() FieldUpdateMessage {
	m.MessageBase = m.MessageBase.Copy()
	m.Field = m.Field.Copy()
	m.Value = copyAny(m.Value)
	return m
}

// FieldValidateMessage is sent by a frontend to request backend validation.
type FieldValidateMessage struct {
	MessageBase
	// Field identifies the field to validate.
	Field FieldRef `json:"field"`
	// Value is the current candidate field value.
	Value any `json:"value"`
}

// Copy returns a deep copy of m.
func (m FieldValidateMessage) Copy() FieldValidateMessage {
	m.MessageBase = m.MessageBase.Copy()
	m.Field = m.Field.Copy()
	m.Value = copyAny(m.Value)
	return m
}

// FormApplyMessage is sent by a frontend when a form block is applied.
type FormApplyMessage struct {
	MessageBase
	// BlockID identifies the form block being applied.
	BlockID string `json:"blockId"`
	// Values contains all current field values in the block.
	Values map[string]any `json:"values"`
}

// Copy returns a deep copy of m.
func (m FormApplyMessage) Copy() FormApplyMessage {
	m.MessageBase = m.MessageBase.Copy()
	m.Values = copyAnyMap(m.Values)
	return m
}

// ButtonPressMessage is sent by a frontend when a declared button is pressed.
type ButtonPressMessage struct {
	MessageBase
	// BlockID identifies the block that owns the button.
	BlockID string `json:"blockId"`
	// ElementPath identifies a nested array element button when present.
	ElementPath []ElementPathSegment `json:"elementPath,omitempty"`
	// ButtonID identifies the button inside the block or element.
	ButtonID string `json:"buttonId"`
}

// Copy returns a deep copy of m.
func (m ButtonPressMessage) Copy() ButtonPressMessage {
	m.MessageBase = m.MessageBase.Copy()
	m.ElementPath = copyElementPath(m.ElementPath)
	return m
}

// AutocompleteRequestMessage is sent by a frontend to request completions.
type AutocompleteRequestMessage struct {
	MessageBase
	// Field identifies the focused field being completed.
	Field FieldRef `json:"field"`
	// Prefix is the current text input prefix.
	Prefix string `json:"prefix"`
}

// Copy returns a deep copy of m.
func (m AutocompleteRequestMessage) Copy() AutocompleteRequestMessage {
	m.MessageBase = m.MessageBase.Copy()
	m.Field = m.Field.Copy()
	return m
}

// Block is a backend-defined menu section with independent generation.
type Block struct {
	// ID is unique within one menu.
	ID string `json:"id"`
	// Order controls frontend display ordering.
	Order int `json:"order"`
	// Generation changes when block structure or item configuration changes.
	Generation uint64 `json:"generation"`
	// Form marks the block as applied as a whole instead of realtime field updates.
	Form bool `json:"form"`
	// Inactive disables all user input for the block.
	Inactive bool `json:"inactive,omitempty"`
	// Collapsible tells the frontend to expose collapse controls.
	Collapsible bool `json:"collapsible,omitempty"`
	// Collapsed is the initial collapsed state unless force-updated.
	Collapsed bool `json:"collapsed,omitempty"`
	// Copyable declares optional text for a frontend copy action.
	Copyable *Copyable `json:"copyable,omitempty"`
	// Items are headers, labels, fields, and buttons in display order.
	Items []Item `json:"items"`
}

// Copy returns a deep copy of b.
func (b Block) Copy() Block {
	b.Copyable = copyCopyablePtr(b.Copyable)
	b.Items = copyItems(b.Items)
	return b
}

// Item is a discriminated menu content item.
type Item struct {
	// Type is one of header, label, progressbar, logs, button, or field.
	Type string `json:"type"`
	// ID is unique inside its owning block or array element.
	ID string `json:"id"`
	// Label is the user-facing caption for fields and buttons.
	Label string `json:"label,omitempty"`
	// Text is the user-facing body for labels and headers.
	Text string `json:"text,omitempty"`
	// Format controls label rendering for plain text, markdown, or code.
	Format string `json:"format,omitempty"`
	// Syntax optionally names a code syntax for highlighted labels.
	Syntax string `json:"syntax,omitempty"`
	// Help is an optional plaintext hint attached to the item.
	Help string `json:"help,omitempty"`
	// Progress is the percentage shown by progressbar items, from 0 to 100.
	Progress *uint `json:"progress,omitempty"`
	// Logs contains lines shown by logs items.
	Logs []LogLine `json:"logs,omitempty"`
	// Inactive disables a button.
	Inactive bool `json:"inactive,omitempty"`
	// Field contains field-specific configuration when Type is field.
	*Field `json:",omitempty"`
}

// Copy returns a deep copy of i.
func (i Item) Copy() Item {
	i.Progress = copyPtr(i.Progress)
	i.Logs = copyLogLines(i.Logs)
	i.Field = copyFieldPtr(i.Field)
	return i
}

// LogLine is one rendered line in a logs item.
type LogLine struct {
	// Level controls the colored prefix shown before Text.
	Level string `json:"level"`
	// Text is the line body.
	Text string `json:"text"`
}

// Copy returns a deep copy of l.
func (l LogLine) Copy() LogLine {
	return l
}

// Field contains input configuration for an Item with type field.
type Field struct {
	// Kind selects the base input widget and value shape.
	Kind string `json:"kind"`
	// Value is the backend-provided default or current value.
	Value any `json:"value,omitempty"`
	// Placeholder is optional empty-state text for text-like inputs.
	Placeholder string `json:"placeholder,omitempty"`
	// Readonly prevents direct user editing when true.
	Readonly bool `json:"readonly,omitempty"`
	// Required means frontend apply is blocked while the value is empty.
	Required bool `json:"required,omitempty"`
	// Validation means frontend should send field.validate messages for changes.
	Validation bool `json:"validate,omitempty"`
	// Status is a backend-provided validation state.
	Status string `json:"status,omitempty"`
	// StatusText is optional explanatory text for Status.
	StatusText string `json:"statusText,omitempty"`
	// Secret requests hidden text entry.
	Secret bool `json:"secret,omitempty"`
	// Multiline allows newline-preserving text input.
	Multiline bool `json:"multiline,omitempty"`
	// Subtype refines text fields, for example email or filepath.
	Subtype string `json:"subtype,omitempty"`
	// Autocomplete configures optional completion requests for text fields.
	Autocomplete *Autocomplete `json:"autocomplete,omitempty"`
	// AllowedValues enumerates values the frontend may offer as choices.
	AllowedValues []any `json:"allowedValues,omitempty"`
	// Min is the optional numeric minimum for int, float, or range fields.
	Min *float64 `json:"min,omitempty"`
	// Max is the optional numeric maximum for int, float, or range fields.
	Max *float64 `json:"max,omitempty"`
	// Fraction is the optional number of fractional digits for floats.
	Fraction *uint `json:"fraction,omitempty"`
	// MaxBytes limits base64 file payload size before encoding.
	MaxBytes *uint64 `json:"maxBytes,omitempty"`
	// Accept lists MIME types accepted by file inputs.
	Accept []string `json:"accept,omitempty"`
	// Templates declares allowed element templates for array fields.
	Templates []ArrayTemplate `json:"templates,omitempty"`
	// Elements is the current array field element snapshot.
	Elements []ArrayElement `json:"elements,omitempty"`
}

// Copy returns a deep copy of f.
func (f Field) Copy() Field {
	f.Value = copyAny(f.Value)
	f.Autocomplete = copyAutocompletePtr(f.Autocomplete)
	f.AllowedValues = copyAnySlice(f.AllowedValues)
	f.Min = copyPtr(f.Min)
	f.Max = copyPtr(f.Max)
	f.Fraction = copyPtr(f.Fraction)
	f.MaxBytes = copyPtr(f.MaxBytes)
	f.Accept = copySlice(f.Accept)
	f.Templates = copyArrayTemplates(f.Templates)
	f.Elements = copyArrayElements(f.Elements)
	return f
}

// Autocomplete configures frontend completion behavior for a field.
type Autocomplete struct {
	// Enabled permits frontend autocomplete.request messages.
	Enabled bool `json:"enabled,omitempty"`
	// Tag is an application-defined completion namespace or hint.
	Tag string `json:"tag,omitempty"`
}

// Copy returns a deep copy of a.
func (a Autocomplete) Copy() Autocomplete {
	return a
}

// ArrayTemplate defines one allowed element shape for an array field.
type ArrayTemplate struct {
	// Name is unique within the owning array field.
	Name string `json:"name"`
	// Label is optional display text for template selection.
	Label string `json:"label,omitempty"`
	// Items are labels, buttons, and fields allowed in elements of this template.
	Items []Item `json:"items"`
}

// Copy returns a deep copy of t.
func (t ArrayTemplate) Copy() ArrayTemplate {
	t.Items = copyItems(t.Items)
	return t
}

// ArrayElement is one concrete element inside an array field.
type ArrayElement struct {
	// ID is unique within the owning array field.
	ID string `json:"id"`
	// Template names the template this element is based on.
	Template string `json:"template"`
	// Items is the element snapshot after backend defaults and status are applied.
	Items []Item `json:"items"`
	// Copyable declares optional text for a frontend copy action.
	Copyable *Copyable `json:"copyable,omitempty"`
}

// Copy returns a deep copy of e.
func (e ArrayElement) Copy() ArrayElement {
	e.Items = copyItems(e.Items)
	e.Copyable = copyCopyablePtr(e.Copyable)
	return e
}

// ArrayElementValue is the wire value shape used when sending array field values.
type ArrayElementValue struct {
	// ID identifies the existing or frontend-created element.
	ID string `json:"id"`
	// Template names the template this value follows.
	Template string `json:"template"`
	// Values contains field values for this element.
	Values map[string]any `json:"values"`
}

// Copy returns a deep copy of v.
func (v ArrayElementValue) Copy() ArrayElementValue {
	v.Values = copyAnyMap(v.Values)
	return v
}

// FieldRef identifies a top-level or nested field in frontend messages.
type FieldRef struct {
	// BlockID identifies the owning top-level block.
	BlockID string `json:"blockId"`
	// FieldID identifies the final field being addressed.
	FieldID string `json:"fieldId"`
	// ElementPath identifies nested array element ownership when present.
	ElementPath []ElementPathSegment `json:"elementPath,omitempty"`
}

// Copy returns a deep copy of r.
func (r FieldRef) Copy() FieldRef {
	r.ElementPath = copyElementPath(r.ElementPath)
	return r
}

// ElementPathSegment identifies one array element nesting step.
type ElementPathSegment struct {
	// ArrayFieldID identifies the array field containing the element.
	ArrayFieldID string `json:"arrayFieldId"`
	// ElementID identifies the array element inside ArrayFieldID.
	ElementID string `json:"elementId"`
}

// Copy returns a deep copy of s.
func (s ElementPathSegment) Copy() ElementPathSegment {
	return s
}

// FieldValue is a named field value used by clients that prefer ordered lists.
type FieldValue struct {
	// FieldID identifies the value owner.
	FieldID string `json:"fieldId"`
	// Value is the current field value.
	Value any `json:"value"`
}

// Copy returns a deep copy of v.
func (v FieldValue) Copy() FieldValue {
	v.Value = copyAny(v.Value)
	return v
}

// Copyable contains clipboard text for copyable blocks or array elements.
type Copyable struct {
	// Text is placed on the system clipboard by frontend copy controls.
	Text string `json:"text"`
}

// Copy returns a deep copy of c.
func (c Copyable) Copy() Copyable {
	return c
}

func copyBlocks(in []Block) []Block {
	if in == nil {
		return nil
	}
	out := make([]Block, len(in))
	for i := range in {
		out[i] = in[i].Copy()
	}
	return out
}

func copyItems(in []Item) []Item {
	if in == nil {
		return nil
	}
	out := make([]Item, len(in))
	for i := range in {
		out[i] = in[i].Copy()
	}
	return out
}

func copyLogLines(in []LogLine) []LogLine {
	if in == nil {
		return nil
	}
	out := make([]LogLine, len(in))
	copy(out, in)
	return out
}

func copyArrayTemplates(in []ArrayTemplate) []ArrayTemplate {
	if in == nil {
		return nil
	}
	out := make([]ArrayTemplate, len(in))
	for i := range in {
		out[i] = in[i].Copy()
	}
	return out
}

func copyArrayElements(in []ArrayElement) []ArrayElement {
	if in == nil {
		return nil
	}
	out := make([]ArrayElement, len(in))
	for i := range in {
		out[i] = in[i].Copy()
	}
	return out
}

func copyElementPath(in []ElementPathSegment) []ElementPathSegment {
	if in == nil {
		return nil
	}
	out := make([]ElementPathSegment, len(in))
	for i := range in {
		out[i] = in[i].Copy()
	}
	return out
}

func copyAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = copyAny(v)
	}
	return out
}

func copyAnySlice(in []any) []any {
	if in == nil {
		return nil
	}
	out := make([]any, len(in))
	for i := range in {
		out[i] = copyAny(in[i])
	}
	return out
}

func copySlice[T any](in []T) []T {
	if in == nil {
		return nil
	}
	out := make([]T, len(in))
	copy(out, in)
	return out
}

func copyPtr[T any](in *T) *T {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}

func copyFieldPtr(in *Field) *Field {
	if in == nil {
		return nil
	}
	out := in.Copy()
	return &out
}

func copyAutocompletePtr(in *Autocomplete) *Autocomplete {
	if in == nil {
		return nil
	}
	out := in.Copy()
	return &out
}

func copyCopyablePtr(in *Copyable) *Copyable {
	if in == nil {
		return nil
	}
	out := in.Copy()
	return &out
}

func copyAny(in any) any {
	switch v := in.(type) {
	case nil:
		return nil
	case map[string]any:
		return copyAnyMap(v)
	case []any:
		return copyAnySlice(v)
	case []ArrayElementValue:
		if v == nil {
			return nil
		}
		out := make([]ArrayElementValue, len(v))
		for i := range v {
			out[i] = v[i].Copy()
		}
		return out
	case ArrayElementValue:
		return v.Copy()
	case []FieldValue:
		if v == nil {
			return nil
		}
		out := make([]FieldValue, len(v))
		for i := range v {
			out[i] = v[i].Copy()
		}
		return out
	case FieldValue:
		return v.Copy()
	default:
		return copyAnyReflect(v)
	}
}

func copyAnyReflect(in any) any {
	value := reflect.ValueOf(in)
	if !value.IsValid() {
		return nil
	}
	copied := copyReflectValue(value)
	if !copied.IsValid() {
		return in
	}
	return copied.Interface()
}

func copyReflectValue(value reflect.Value) reflect.Value {
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		copied := copyReflectValue(value.Elem())
		if !copied.IsValid() {
			return value
		}
		if copied.Type().AssignableTo(value.Type()) {
			return copied
		}
		out := reflect.New(value.Type()).Elem()
		out.Set(copied)
		return out
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		elem := copyReflectValue(value.Elem())
		if !elem.IsValid() {
			return value
		}
		out := reflect.New(value.Type().Elem())
		out.Elem().Set(elem)
		return out
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			out.Index(i).Set(copyReflectValue(value.Index(i)))
		}
		return out
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		out := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			out.SetMapIndex(copyReflectValue(iter.Key()), copyReflectValue(iter.Value()))
		}
		return out
	default:
		return value
	}
}
