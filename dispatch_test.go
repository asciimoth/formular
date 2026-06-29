// nolint
package formular

import (
	"errors"
	"reflect"
	"testing"
)

func TestDispatchFormMessageRoutesMatchingMessages(t *testing.T) {
	var calls []string
	handler := FormHandler{
		MenuID:  "settings",
		BlockID: "profile",
		OnUpdate: func(fieldID string, value any) error {
			calls = append(calls, "update:"+fieldID+":"+value.(string))
			return nil
		},
		OnValidate: func(fieldID string, value any) error {
			calls = append(calls, "validate:"+fieldID+":"+value.(string))
			return nil
		},
		OnApply: func(fieldID string, value any) error {
			calls = append(calls, "apply:"+fieldID+":"+value.(string))
			return nil
		},
	}

	handled, err := DispatchFormMessage(FieldUpdateMessage{
		MessageBase: MessageBase{Type: MessageFieldUpdate, MenuID: "settings"},
		Field:       FieldRef{BlockID: "profile", FieldID: "name"},
		Value:       "Ada",
	}, handler)
	if err != nil || !handled {
		t.Fatalf("field.update handled = %v, err = %v", handled, err)
	}

	handled, err = DispatchFormMessage([]byte(`{"type":"field.validate","menuId":"settings","field":{"blockId":"profile","fieldId":"email"},"value":"ada@example.com"}`), handler)
	if err != nil || !handled {
		t.Fatalf("field.validate handled = %v, err = %v", handled, err)
	}

	handled, err = DispatchFormMessage(FormApplyMessage{
		MessageBase: MessageBase{Type: MessageFormApply, MenuID: "settings"},
		BlockID:     "profile",
		Values:      map[string]any{"email": "ada@example.com", "name": "Ada"},
	}, handler)
	if err != nil || !handled {
		t.Fatalf("form.apply handled = %v, err = %v", handled, err)
	}

	want := []string{
		"update:name:Ada",
		"validate:email:ada@example.com",
		"apply:email:ada@example.com",
		"apply:name:Ada",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestDispatchFormMessageIgnoresMismatchesAndReportsErrors(t *testing.T) {
	handler := FormHandler{
		MenuID:  "settings",
		BlockID: "profile",
		OnUpdate: func(fieldID string, value any) error {
			return errors.New("boom")
		},
	}

	handled, err := DispatchFormMessage(FieldUpdateMessage{
		MessageBase: MessageBase{Type: MessageFieldUpdate, MenuID: "other"},
		Field:       FieldRef{BlockID: "profile", FieldID: "name"},
		Value:       "Ada",
	}, handler)
	if err != nil || handled {
		t.Fatalf("mismatched menu handled = %v, err = %v", handled, err)
	}

	handled, err = DispatchFormMessage([]byte(`{"type":"button.press","menuId":"settings","blockId":"profile","buttonId":"save"}`), handler)
	if err != nil || handled {
		t.Fatalf("unsupported message handled = %v, err = %v", handled, err)
	}

	handled, err = DispatchFormMessage(FieldUpdateMessage{
		MessageBase: MessageBase{Type: MessageFieldUpdate, MenuID: "settings"},
		Field:       FieldRef{BlockID: "profile", FieldID: "name"},
		Value:       "Ada",
	}, handler)
	if err == nil || !handled {
		t.Fatalf("callback error handled = %v, err = %v", handled, err)
	}
}
