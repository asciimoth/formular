// nolint
package formular

import "testing"

func TestFieldBuildersApplyOptions(t *testing.T) {
	item := TextField(
		"email",
		"Email",
		"ada@example.com",
		Required,
		Readonly,
		Validation,
		Placeholder("name@example.com"),
		Help("Account email"),
		Status(StatusOK),
		StatusText("Looks good"),
		AllowedValues("ada@example.com", "grace@example.com"),
		AutocompleteConfig(Autocomplete{Enabled: true, Tag: "email"}),
	)

	if item.Type != ItemField || item.ID != "email" || item.Label != "Email" || item.Help != "Account email" {
		t.Fatalf("unexpected field item: %+v", item)
	}
	field := item.Field
	if field.Kind != FieldText || field.Value != "ada@example.com" || !field.Required || !field.Readonly || !field.Validation {
		t.Fatalf("unexpected text field: %+v", field)
	}
	if field.Placeholder != "name@example.com" || field.Status != StatusOK || field.StatusText != "Looks good" {
		t.Fatalf("unexpected field metadata: %+v", field)
	}
	if field.Autocomplete == nil || field.Autocomplete.Tag != "email" {
		t.Fatalf("missing autocomplete: %+v", field.Autocomplete)
	}
	if got := field.AllowedValues[1]; got != "grace@example.com" {
		t.Fatalf("allowed value = %v, want grace@example.com", got)
	}
	if err := item.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSpecificFieldAndItemBuilders(t *testing.T) {
	items := []Item{
		IntField("port", "Port", 8080),
		FloatField("ratio", "Ratio", 0.5),
		CheckboxField("enabled", "Enabled", true),
		RadioField("mode", "Mode", "manual", []Choice{{Value: "auto"}, {Value: "manual", Label: "Manual"}}),
		ReadonlyTextField("path", "Path", "/tmp/file"),
		SecretTextField("token", "Token", "secret"),
		MultilineTextField("notes", "Notes", "line 1\nline 2"),
		Button("save", "Save", Help("Persist settings")),
		PlainLabel("summary", "Ready"),
		Logs("logs", "Logs", []LogLine{{Level: LogInfo, Text: "started"}}),
	}

	if items[0].Field.Kind != FieldInt || items[0].Field.Value != 8080 {
		t.Fatalf("unexpected int field: %+v", items[0])
	}
	if items[1].Field.Kind != FieldFloat || items[1].Field.Value != 0.5 {
		t.Fatalf("unexpected float field: %+v", items[1])
	}
	if items[2].Field.Kind != FieldCheckbox || items[2].Field.Value != true {
		t.Fatalf("unexpected checkbox field: %+v", items[2])
	}
	if got := items[3].Field.AllowedValues; len(got) != 2 || got[1] != "manual" {
		t.Fatalf("unexpected radio choices: %+v", got)
	}
	if !items[4].Field.Readonly || !items[5].Field.Secret || !items[6].Field.Multiline {
		t.Fatalf("specialized text builders did not set expected flags: %+v %+v %+v", items[4], items[5], items[6])
	}
	if items[7].Type != ItemButton || items[7].Help != "Persist settings" {
		t.Fatalf("unexpected button: %+v", items[7])
	}
	if items[8].Format != TextPlain || items[9].Logs[0].Text != "started" {
		t.Fatalf("unexpected display items: %+v %+v", items[8], items[9])
	}
	for _, item := range items {
		if err := item.Validate(); err != nil {
			t.Fatalf("%s did not validate: %v", item.ID, err)
		}
	}
}

func TestBuilderCopiesMutableInputs(t *testing.T) {
	allowed := map[string]any{"name": "Ada"}
	item := TextField("name", "Name", "Ada", AllowedValues(allowed))
	allowed["name"] = "Grace"
	if got := item.Field.AllowedValues[0].(map[string]any)["name"]; got != "Ada" {
		t.Fatalf("allowed value shared caller map: %v", got)
	}

	lines := []LogLine{{Level: LogInfo, Text: "ready"}}
	logs := Logs("logs", "Logs", lines)
	lines[0].Text = "changed"
	if logs.Logs[0].Text != "ready" {
		t.Fatal("logs item shared caller slice")
	}
}
