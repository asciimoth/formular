// nolint
package formular

import (
	"encoding/json"
	"math"
	"testing"
)

func TestScalarValueDecoders(t *testing.T) {
	if got, ok := StringValue("Ada"); !ok || got != "Ada" {
		t.Fatalf("StringValue = %q, %v", got, ok)
	}
	if _, ok := StringValue(42); ok {
		t.Fatal("StringValue should reject non-strings")
	}

	for _, input := range []any{42, int8(42), uint16(42), json.Number("42"), float64(42), float32(42)} {
		got, ok := IntValue(input)
		if !ok || got != 42 {
			t.Fatalf("IntValue(%T(%v)) = %d, %v", input, input, got, ok)
		}
	}
	for _, input := range []any{float64(42.5), math.NaN(), uint64(math.MaxUint64)} {
		if got, ok := IntValue(input); ok {
			t.Fatalf("IntValue(%T(%v)) = %d, true; want false", input, input, got)
		}
	}

	for _, input := range []any{42, uint(42), json.Number("42.5"), float32(42.5), float64(42.5)} {
		got, ok := FloatValue(input)
		if !ok || got == 0 {
			t.Fatalf("FloatValue(%T(%v)) = %v, %v", input, input, got, ok)
		}
	}
	if _, ok := FloatValue(math.Inf(1)); ok {
		t.Fatal("FloatValue should reject infinity")
	}

	if got, ok := BoolValue(true); !ok || !got {
		t.Fatalf("BoolValue(true) = %v, %v", got, ok)
	}
	if _, ok := BoolValue("true"); ok {
		t.Fatal("BoolValue should reject string booleans")
	}
}

func TestArrayElementValueDecoders(t *testing.T) {
	var raw any
	if err := json.Unmarshal([]byte(`{"id":"db-1","template":"database","values":{"host":"localhost","port":5432,"enabled":true}}`), &raw); err != nil {
		t.Fatal(err)
	}

	element, ok := ArrayElementValueFromAny(raw)
	if !ok {
		t.Fatal("expected JSON-like map to decode")
	}
	if element.ID != "db-1" || element.Template != "database" {
		t.Fatalf("unexpected element identity: %+v", element)
	}
	if host, ok := ArrayElementStringValue(element, "host"); !ok || host != "localhost" {
		t.Fatalf("host = %q, %v", host, ok)
	}
	if port, ok := ArrayElementIntValue(element, "port"); !ok || port != 5432 {
		t.Fatalf("port = %d, %v", port, ok)
	}
	if enabled, ok := ArrayElementBoolValue(element, "enabled"); !ok || !enabled {
		t.Fatalf("enabled = %v, %v", enabled, ok)
	}

	element.Values["host"] = "changed"
	copied, ok := ArrayElementValueFromAny(raw)
	if !ok || copied.Values["host"] != "localhost" {
		t.Fatal("decoded element should not share value maps")
	}

	for _, input := range []any{
		map[string]any{"template": "database", "values": map[string]any{}},
		map[string]any{"id": "db-1", "values": map[string]any{}},
		map[string]any{"id": "db-1", "template": "database", "values": "bad"},
	} {
		if _, ok := ArrayElementValueFromAny(input); ok {
			t.Fatalf("ArrayElementValueFromAny(%v) should fail", input)
		}
	}
}

func TestArrayElementValuesFromAny(t *testing.T) {
	input := []any{
		map[string]any{"id": "one", "template": "row", "values": map[string]any{"name": "one"}},
		ArrayElementValue{ID: "two", Template: "row", Values: map[string]any{"name": "two"}},
	}
	values, ok := ArrayElementValuesFromAny(input)
	if !ok || len(values) != 2 {
		t.Fatalf("ArrayElementValuesFromAny len = %d, %v", len(values), ok)
	}
	if values[0].Values["name"] != "one" || values[1].Values["name"] != "two" {
		t.Fatalf("unexpected decoded values: %+v", values)
	}
	if _, ok := ArrayElementValuesFromAny([]any{map[string]any{"id": "missing-template"}}); ok {
		t.Fatal("malformed array element slice should fail")
	}
}

func TestInstantiateArrayTemplateCopiesTemplateAndOverlaysValues(t *testing.T) {
	template := ArrayTemplate{
		Name: "database",
		Items: []Item{
			TextField("host", "Host", "localhost", Required, Status(StatusOK)),
			IntField("port", "Port", 5432),
			PlainLabel("hint", "Use a reachable host"),
		},
	}

	element := InstantiateArrayTemplate(template, "db-1", map[string]any{"host": "db.internal"})
	if element.ID != "db-1" || element.Template != "database" {
		t.Fatalf("unexpected element identity: %+v", element)
	}
	if got := element.Items[0].Field.Value; got != "db.internal" {
		t.Fatalf("host = %v, want db.internal", got)
	}
	if got := element.Items[1].Field.Value; got != 5432 {
		t.Fatalf("port = %v, want template default", got)
	}
	if !element.Items[0].Field.Required || element.Items[0].Field.Status != StatusOK {
		t.Fatalf("field metadata was not preserved: %+v", element.Items[0].Field)
	}

	element.Items[0].Field.Value = "changed"
	if got := template.Items[0].Field.Value; got != "localhost" {
		t.Fatalf("template mutated through element: %v", got)
	}
}
