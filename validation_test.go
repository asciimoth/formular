// nolint
package formular

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateAcceptsValidMenuSnapshot(t *testing.T) {
	maxBytes := uint64(1024)
	menu := MenuSnapshotMessage{
		MessageBase: MessageBase{
			Type:           MessageMenuSnapshot,
			MenuID:         "settings",
			MenuGeneration: 1,
		},
		Blocks: []Block{
			{
				ID:         "account",
				Order:      10,
				Generation: 1,
				Form:       true,
				Items: []Item{
					{Type: ItemHeader, ID: "title", Text: "Account"},
					{
						Type:  ItemField,
						ID:    "email",
						Label: "Email",
						Field: &Field{
							Kind:       FieldText,
							Subtype:    "email",
							Validation: true,
							Status:     StatusOK,
						},
					},
					{
						Type:  ItemField,
						ID:    "avatar",
						Label: "Avatar",
						Field: &Field{
							Kind:     FieldFile,
							MaxBytes: &maxBytes,
							Accept:   []string{"image/png"},
						},
					},
					{
						Type:  ItemField,
						ID:    "credentials",
						Label: "Credentials",
						Field: &Field{
							Kind: FieldArray,
							Templates: []ArrayTemplate{
								{
									Name: "token",
									Items: []Item{
										{Type: ItemField, ID: "value", Label: "Value", Field: &Field{Kind: FieldText}},
									},
								},
							},
							Elements: []ArrayElement{
								{
									ID:       "token-1",
									Template: "token",
									Items: []Item{
										{Type: ItemField, ID: "value", Label: "Value", Field: &Field{Kind: FieldText}},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := menu.Validate(); err != nil {
		t.Fatalf("valid menu failed validation: %v", err)
	}
}

func TestValidateRejectsInvalidMenuSnapshot(t *testing.T) {
	menu := MenuSnapshotMessage{
		MessageBase: MessageBase{
			Type:   MessageBlockSnapshot,
			MenuID: "settings",
		},
		Blocks: []Block{
			{
				ID:         "dup",
				Generation: 1,
				Items: []Item{
					{Type: ItemField, ID: "broken", Label: "Broken", Field: &Field{Kind: FieldFile, Validation: true}},
					{Type: ItemField, ID: "broken", Label: "Broken Again", Field: &Field{Kind: FieldRadio}},
				},
			},
			{
				ID:         "dup",
				Generation: 2,
				Items:      []Item{},
			},
		},
	}

	err := menu.Validate()
	if err == nil {
		t.Fatal("invalid menu passed validation")
	}
	if !IsValidationError(err) {
		t.Fatal("expected IsValidationError to report true")
	}
	var validationErrs ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatal("expected ValidationErrors")
	}
	if len(validationErrs) < 4 {
		t.Fatalf("expected multiple validation errors, got %d: %v", len(validationErrs), err)
	}

	text := err.Error()
	for _, want := range []string{
		`message.type: must be "menu.snapshot"`,
		"duplicates message.blocks[0].id",
		"validate: must be false for file fields",
		"allowedValues: must contain at least one value for radio fields",
		"duplicates message.blocks[0].items[0].id",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing validation message %q in %q", want, text)
		}
	}
}

func TestValidateRejectsInvalidArrayField(t *testing.T) {
	field := Field{
		Kind: FieldArray,
		Templates: []ArrayTemplate{
			{
				Name: "row",
				Items: []Item{
					{Type: ItemHeader, ID: "title", Text: "Not Allowed"},
				},
			},
		},
		Elements: []ArrayElement{
			{
				ID:       "row-1",
				Template: "missing",
				Items: []Item{
					{Type: ItemField, ID: "nested", Label: "Nested", Field: &Field{Kind: FieldArray}},
				},
			},
		},
	}

	err := field.Validate()
	if err == nil {
		t.Fatal("invalid array field passed validation")
	}
	text := err.Error()
	for _, want := range []string{
		"headers are not allowed in array elements",
		"must match one of the array field templates",
		"array fields are not allowed in array elements",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing validation message %q in %q", want, text)
		}
	}
}

func TestValidateRejectsInvalidValues(t *testing.T) {
	update := FieldUpdateMessage{
		MessageBase: MessageBase{Type: MessageFieldUpdate, MenuID: "settings"},
		Field:       FieldRef{BlockID: "account", FieldID: "email"},
		Value:       map[string]any{"object": "not a field value"},
	}
	if err := update.Validate(); err == nil || !strings.Contains(err.Error(), "objects are not valid field values") {
		t.Fatalf("expected object value validation error, got %v", err)
	}

	apply := FormApplyMessage{
		MessageBase: MessageBase{Type: MessageFormApply, MenuID: "settings"},
		BlockID:     "account",
		Values: map[string]any{
			"credentials": []any{
				map[string]any{
					"id":       "cred-1",
					"template": "token",
					"values":   map[string]any{"token": "secret"},
				},
			},
		},
	}
	if err := apply.Validate(); err != nil {
		t.Fatalf("valid form values failed validation: %v", err)
	}
}

func TestValidateFieldValueKinds(t *testing.T) {
	tests := []struct {
		name  string
		field Field
		want  string
	}{
		{
			name:  "text rejects non-string",
			field: Field{Kind: FieldText, Value: 42},
			want:  "must be a string for text fields",
		},
		{
			name:  "int rejects fractional value",
			field: Field{Kind: FieldInt, Value: 1.5},
			want:  "must be an integer for int fields",
		},
		{
			name:  "file rejects non-base64",
			field: Field{Kind: FieldFile, Value: "not base64"},
			want:  "must be valid base64 for file fields",
		},
		{
			name:  "checkbox rejects string",
			field: Field{Kind: FieldCheckbox, Value: "true"},
			want:  "must be a boolean for checkbox fields",
		},
		{
			name:  "radio requires selected allowed value",
			field: Field{Kind: FieldRadio, Value: "c", AllowedValues: []any{"a", "b"}},
			want:  "must match one of allowedValues for radio fields",
		},
		{
			name:  "array rejects scalar",
			field: Field{Kind: FieldArray, Value: "not array"},
			want:  "must be an array of array element values",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.field.Validate()
			if err == nil {
				t.Fatal("invalid field passed validation")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("missing validation message %q in %q", tt.want, err.Error())
			}
		})
	}

	valid := []Field{
		{Kind: FieldText, Value: "hello"},
		{Kind: FieldInt, Value: float64(3)},
		{Kind: FieldFloat, Value: 3.14},
		{Kind: FieldFile, Value: "aGVsbG8="},
		{Kind: FieldCheckbox, Value: true},
		{Kind: FieldRadio, Value: "a", AllowedValues: []any{"a", "b"}},
		{Kind: FieldArray, Value: []ArrayElementValue{{ID: "row-1", Template: "row", Values: map[string]any{"name": "value"}}}},
	}
	for _, field := range valid {
		if err := field.Validate(); err != nil {
			t.Fatalf("valid field failed validation: %#v: %v", field, err)
		}
	}
}
