package formular

import (
	"encoding/json"
	"math"
	"reflect"
)

// StringValue decodes a Formular scalar value as a string.
//
// Only concrete strings are accepted. Failed conversions return "", false.
func StringValue(v any) (string, bool) {
	out, ok := v.(string)
	return out, ok
}

// IntValue decodes a Formular scalar value as an int.
//
// Signed and unsigned integer types are accepted when they fit in int.
// json.Number and floating-point values are accepted only when finite and
// exactly whole-numbered. Failed conversions return 0, false.
func IntValue(v any) (int, bool) {
	switch typed := v.(type) {
	case int:
		return typed, true
	case int8:
		return int(typed), true
	case int16:
		return int(typed), true
	case int32:
		return int(typed), true
	case int64:
		return intFromInt64(typed)
	case uint:
		return intFromUint64(uint64(typed))
	case uint8:
		return int(typed), true
	case uint16:
		return int(typed), true
	case uint32:
		return intFromUint64(uint64(typed))
	case uint64:
		return intFromUint64(typed)
	case json.Number:
		if i, err := typed.Int64(); err == nil {
			return intFromInt64(i)
		}
		f, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return intFromFloat64(f)
	case float32:
		return intFromFloat64(float64(typed))
	case float64:
		return intFromFloat64(typed)
	default:
		return 0, false
	}
}

// FloatValue decodes a Formular scalar value as a finite float64.
//
// Go integer and floating-point numeric types, plus json.Number, are accepted.
// Failed conversions return 0, false.
func FloatValue(v any) (float64, bool) {
	switch typed := v.(type) {
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case json.Number:
		f, err := typed.Float64()
		if err != nil || !isFinite(f) {
			return 0, false
		}
		return f, true
	case float32:
		f := float64(typed)
		if !isFinite(f) {
			return 0, false
		}
		return f, true
	case float64:
		if !isFinite(typed) {
			return 0, false
		}
		return typed, true
	default:
		return 0, false
	}
}

// BoolValue decodes a Formular scalar value as a bool.
//
// Only concrete booleans are accepted. Failed conversions return false, false.
func BoolValue(v any) (bool, bool) {
	out, ok := v.(bool)
	return out, ok
}

// ArrayElementValueFromAny decodes a single array element value from typed or
// JSON-like data.
//
// The canonical ArrayElementValue shape is accepted directly. Map-shaped values
// must contain non-empty string "id" and "template" fields. Missing "values"
// is treated as an empty value map; malformed input returns false.
func ArrayElementValueFromAny(v any) (ArrayElementValue, bool) {
	switch typed := v.(type) {
	case ArrayElementValue:
		return typed.Copy(), true
	case *ArrayElementValue:
		if typed == nil {
			return ArrayElementValue{}, false
		}
		return typed.Copy(), true
	case map[string]any:
		return arrayElementValueFromMap(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ArrayElementValue{}, false
		}
		var out ArrayElementValue
		if err := json.Unmarshal(data, &out); err != nil {
			return ArrayElementValue{}, false
		}
		if out.ID == "" || out.Template == "" {
			return ArrayElementValue{}, false
		}
		if out.Values == nil {
			out.Values = map[string]any{}
		}
		return out.Copy(), true
	}
}

// ArrayElementValuesFromAny decodes one or more array element values.
//
// It accepts []ArrayElementValue, []any, and other slice/array values whose
// elements individually decode with ArrayElementValueFromAny. Passing a single
// element value returns a one-element slice.
func ArrayElementValuesFromAny(v any) ([]ArrayElementValue, bool) {
	switch typed := v.(type) {
	case nil:
		return nil, false
	case []ArrayElementValue:
		out := make([]ArrayElementValue, len(typed))
		for i := range typed {
			out[i] = typed[i].Copy()
		}
		return out, true
	case []any:
		return arrayElementValuesFromSlice(typed)
	}

	value := reflect.ValueOf(v)
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return nil, false
		}
		value = value.Elem()
	}
	if value.IsValid() && (value.Kind() == reflect.Slice || value.Kind() == reflect.Array) {
		out := make([]ArrayElementValue, 0, value.Len())
		for i := 0; i < value.Len(); i++ {
			element, ok := ArrayElementValueFromAny(value.Index(i).Interface())
			if !ok {
				return nil, false
			}
			out = append(out, element)
		}
		return out, true
	}

	element, ok := ArrayElementValueFromAny(v)
	if !ok {
		return nil, false
	}
	return []ArrayElementValue{element}, true
}

// ArrayElementFieldValue returns one field value from an array element value.
func ArrayElementFieldValue(v any, fieldID string) (any, bool) {
	if fieldID == "" {
		return nil, false
	}
	element, ok := ArrayElementValueFromAny(v)
	if !ok {
		return nil, false
	}
	value, ok := element.Values[fieldID]
	if !ok {
		return nil, false
	}
	return copyAny(value), true
}

// ArrayElementStringValue returns one string field value from an array element.
func ArrayElementStringValue(v any, fieldID string) (string, bool) {
	value, ok := ArrayElementFieldValue(v, fieldID)
	if !ok {
		return "", false
	}
	return StringValue(value)
}

// ArrayElementIntValue returns one integer field value from an array element.
func ArrayElementIntValue(v any, fieldID string) (int, bool) {
	value, ok := ArrayElementFieldValue(v, fieldID)
	if !ok {
		return 0, false
	}
	return IntValue(value)
}

// ArrayElementBoolValue returns one boolean field value from an array element.
func ArrayElementBoolValue(v any, fieldID string) (bool, bool) {
	value, ok := ArrayElementFieldValue(v, fieldID)
	if !ok {
		return false, false
	}
	return BoolValue(value)
}

// InstantiateArrayTemplate copies an array template into a concrete element and
// overlays field values by item ID.
//
// Unspecified fields keep their template defaults. The template is not mutated.
func InstantiateArrayTemplate(template ArrayTemplate, id string, values map[string]any) ArrayElement {
	element := ArrayElement{
		ID:       id,
		Template: template.Name,
		Items:    copyItems(template.Items),
	}
	applyValuesToItems(element.Items, values)
	return element
}

func arrayElementValueFromMap(raw map[string]any) (ArrayElementValue, bool) {
	id, ok := raw["id"].(string)
	if !ok || id == "" {
		return ArrayElementValue{}, false
	}
	template, ok := raw["template"].(string)
	if !ok || template == "" {
		return ArrayElementValue{}, false
	}
	values, ok := anyMapFromAny(raw["values"])
	if !ok {
		return ArrayElementValue{}, false
	}
	return ArrayElementValue{ID: id, Template: template, Values: values}.Copy(), true
}

func anyMapFromAny(v any) (map[string]any, bool) {
	if v == nil {
		return map[string]any{}, true
	}
	if values, ok := v.(map[string]any); ok {
		return copyAnyMap(values), true
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, false
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, false
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, true
}

func arrayElementValuesFromSlice(raw []any) ([]ArrayElementValue, bool) {
	out := make([]ArrayElementValue, 0, len(raw))
	for _, item := range raw {
		value, ok := ArrayElementValueFromAny(item)
		if !ok {
			return nil, false
		}
		out = append(out, value)
	}
	return out, true
}

func intFromInt64(v int64) (int, bool) {
	out := int(v)
	if int64(out) != v {
		return 0, false
	}
	return out, true
}

func intFromUint64(v uint64) (int, bool) {
	out := int(v)
	if out < 0 || uint64(out) != v {
		return 0, false
	}
	return out, true
}

func intFromFloat64(v float64) (int, bool) {
	if !isFinite(v) || math.Trunc(v) != v {
		return 0, false
	}
	out := int(v)
	if float64(out) != v {
		return 0, false
	}
	return out, true
}
