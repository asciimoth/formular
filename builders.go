package formular

// FieldOption mutates a field item produced by the field builders in this file.
//
// The option receives the full item so it can set item-level metadata such as
// Help as well as field-level metadata. Field-only options are no-ops when used
// with non-field builders such as Button.
type FieldOption = func(*Item)

// ItemOption mutates an item produced by item builders such as Button.
type ItemOption = func(*Item)

// Choice describes one radio field choice.
//
// The current Formular protocol transmits allowed radio values but does not
// carry separate display labels for each value. Label is retained for future
// callers that want to keep local choice metadata near the builder call.
type Choice struct {
	// Value is the protocol value placed in the radio field's allowedValues.
	Value string
	// Label is optional caller-owned display metadata and is not serialized by
	// the core protocol today.
	Label string
}

// Readonly marks a field as display-only.
var Readonly FieldOption = func(item *Item) {
	withField(item, func(field *Field) {
		field.Readonly = true
	})
}

// Required marks a field as required for frontend form apply.
var Required FieldOption = func(item *Item) {
	withField(item, func(field *Field) {
		field.Required = true
	})
}

// Secret requests hidden entry for text fields.
var Secret FieldOption = func(item *Item) {
	withField(item, func(field *Field) {
		field.Secret = true
	})
}

// Multiline allows newline-preserving entry for text fields.
var Multiline FieldOption = func(item *Item) {
	withField(item, func(field *Field) {
		field.Multiline = true
	})
}

// Placeholder sets placeholder text for a field.
func Placeholder(text string) FieldOption {
	return func(item *Item) {
		withField(item, func(field *Field) {
			field.Placeholder = text
		})
	}
}

// Help sets plaintext help on any item.
func Help(text string) ItemOption {
	return func(item *Item) {
		item.Help = text
	}
}

// Validation enables frontend field.validate messages for a field.
var Validation FieldOption = func(item *Item) {
	withField(item, func(field *Field) {
		field.Validation = true
	})
}

// Status sets the backend validation status shown for a field.
func Status(status string) FieldOption {
	return func(item *Item) {
		withField(item, func(field *Field) {
			field.Status = status
		})
	}
}

// StatusText sets explanatory text for a field status.
func StatusText(text string) FieldOption {
	return func(item *Item) {
		withField(item, func(field *Field) {
			field.StatusText = text
		})
	}
}

// AllowedValues sets the protocol allowedValues list for text, int, float, or
// radio fields. Values are deep-copied before being stored on the field.
func AllowedValues(values ...any) FieldOption {
	return func(item *Item) {
		withField(item, func(field *Field) {
			field.AllowedValues = copyAnySlice(values)
		})
	}
}

// AutocompleteConfig sets text-field autocomplete behavior.
//
// The name avoids colliding with the Autocomplete struct type.
func AutocompleteConfig(config Autocomplete) FieldOption {
	return func(item *Item) {
		withField(item, func(field *Field) {
			field.Autocomplete = copyAutocompletePtr(&config)
		})
	}
}

// TextField constructs a text field item.
func TextField(id, label, value string, opts ...FieldOption) Item {
	return fieldItem(id, label, Field{Kind: FieldText, Value: value}, opts...)
}

// IntField constructs an integer field item.
func IntField(id, label string, value int, opts ...FieldOption) Item {
	return fieldItem(id, label, Field{Kind: FieldInt, Value: value}, opts...)
}

// FloatField constructs a floating-point field item.
func FloatField(id, label string, value float64, opts ...FieldOption) Item {
	return fieldItem(id, label, Field{Kind: FieldFloat, Value: value}, opts...)
}

// CheckboxField constructs a checkbox field item.
func CheckboxField(id, label string, value bool, opts ...FieldOption) Item {
	return fieldItem(id, label, Field{Kind: FieldCheckbox, Value: value}, opts...)
}

// RadioField constructs a radio field item from string choices.
func RadioField(id, label string, value string, options []Choice, opts ...FieldOption) Item {
	allowed := make([]any, len(options))
	for i := range options {
		allowed[i] = options[i].Value
	}
	base := fieldItem(id, label, Field{Kind: FieldRadio, Value: value, AllowedValues: allowed}, opts...)
	return base
}

// ReadonlyTextField constructs a readonly text field item.
func ReadonlyTextField(id, label, value string, opts ...FieldOption) Item {
	return TextField(id, label, value, append([]FieldOption{Readonly}, opts...)...)
}

// SecretTextField constructs a text field item that requests hidden entry.
func SecretTextField(id, label, value string, opts ...FieldOption) Item {
	return TextField(id, label, value, append([]FieldOption{Secret}, opts...)...)
}

// MultilineTextField constructs a text field item that preserves newlines.
func MultilineTextField(id, label, value string, opts ...FieldOption) Item {
	return TextField(id, label, value, append([]FieldOption{Multiline}, opts...)...)
}

// Button constructs a button item.
func Button(id, label string, opts ...ItemOption) Item {
	item := Item{Type: ItemButton, ID: id, Label: label}
	applyItemOptions(&item, opts)
	return item
}

// PlainLabel constructs a plain text label item.
func PlainLabel(id, text string) Item {
	return Item{Type: ItemLabel, ID: id, Text: text, Format: TextPlain}
}

// Logs constructs a logs display item.
func Logs(id, label string, lines []LogLine) Item {
	return Item{Type: ItemLogs, ID: id, Label: label, Logs: copyLogLines(lines)}
}

func fieldItem(id, label string, field Field, opts ...FieldOption) Item {
	copied := field.Copy()
	item := Item{Type: ItemField, ID: id, Label: label, Field: &copied}
	applyItemOptions(&item, opts)
	return item
}

func applyItemOptions(item *Item, opts []func(*Item)) {
	for _, opt := range opts {
		if opt != nil {
			opt(item)
		}
	}
}

func withField(item *Item, update func(*Field)) {
	if item == nil || item.Field == nil {
		return
	}
	update(item.Field)
}
