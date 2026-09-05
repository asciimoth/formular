// nolint
package formular

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestDialogConstructorsAndCopies(t *testing.T) {
	image := DialogResourceFromBytes("challenge", "image/png", "Captcha challenge", []byte("png data"))
	dialog := CaptchaDialog("sign-in", "Verify", "Enter the text in the image.", image)
	dialog.Placeholder = "Text from image"
	message := DialogCreate("account", 7, dialog)

	dialog.Resources[0].Data = "changed"
	if message.Type != MessageDialogCreate || message.MenuID != "account" || message.MenuGeneration != 7 {
		t.Fatalf("unexpected dialog envelope: %+v", message.MessageBase)
	}
	if message.Dialog.Resources[0].Data != "cG5nIGRhdGE=" {
		t.Fatal("dialog creation shared resource state")
	}
	if err := message.Validate(); err != nil {
		t.Fatalf("valid captcha dialog failed validation: %v", err)
	}

	options := []DialogOption{{Value: "small", Label: "Small", Selected: true}, {Value: "large", Label: "Large"}}
	selection := SelectionDialog("size", "Size", "Select a size.", options...)
	options[0].Label = "changed"
	if selection.Options[0].Label != "Small" {
		t.Fatal("selection dialog shared caller options")
	}
	if err := selection.Validate(); err != nil {
		t.Fatalf("valid selection dialog failed validation: %v", err)
	}
	if err := YesNoDialog("delete", "Delete item?", "This cannot be undone.").Validate(); err != nil {
		t.Fatalf("valid yes/no dialog failed validation: %v", err)
	}

	response := DialogResponseMessage{
		MessageBase: MessageBase{Type: MessageDialogResponse, MenuID: "account", MenuGeneration: 7},
		DialogID:    "size",
		Value:       []string{"small", "large"},
	}
	copy := response.Copy()
	copy.Value.([]string)[0] = "changed"
	if response.Value.([]string)[0] != "small" {
		t.Fatal("dialog response shared result state")
	}
	if err := response.Validate(); err != nil {
		t.Fatalf("valid dialog response failed validation: %v", err)
	}

	no := DialogResponseMessage{
		MessageBase: MessageBase{Type: MessageDialogResponse, MenuID: "account"},
		DialogID:    "delete",
		Value:       false,
	}
	data, err := json.Marshal(no)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\"value\":false") {
		t.Fatalf("false dialog result was omitted: %s", data)
	}
}

func TestDialogValidationRejectsInvalidKindsAndPayloads(t *testing.T) {
	tests := []struct {
		name   string
		dialog Dialog
		want   string
	}{
		{
			name:   "selection needs options",
			dialog: Dialog{ID: "choice", Kind: DialogKindSelection, Title: "Choose"},
			want:   "must contain at least one option",
		},
		{
			name: "single selection has one default",
			dialog: Dialog{ID: "choice", Kind: DialogKindSelection, Title: "Choose", Options: []DialogOption{
				{Value: "a", Label: "A", Selected: true},
				{Value: "b", Label: "B", Selected: true},
			}},
			want: "more than one selected option",
		},
		{
			name: "captcha needs image",
			dialog: Dialog{ID: "challenge", Kind: DialogKindCaptcha, Title: "Verify", Resources: []DialogResource{
				DialogResourceFromBytes("sound", "audio/ogg", "Audio challenge", []byte("sound")),
			}},
			want: "at least one image",
		},
		{
			name: "resource needs base64",
			dialog: Dialog{ID: "challenge", Kind: DialogKindCaptcha, Title: "Verify", Resources: []DialogResource{
				{ID: "image", MIMEType: "image/png", Data: "not base64"},
			}},
			want: "must be valid base64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dialog.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validation error = %v, want text %q", err, tt.want)
			}
		})
	}

	invalidResponse := DialogResponseMessage{
		MessageBase: MessageBase{Type: MessageDialogResponse, MenuID: "account"},
		DialogID:    "choice",
		Value:       42,
	}
	if err := invalidResponse.Validate(); err == nil || !strings.Contains(err.Error(), "boolean, string, string array, or null") {
		t.Fatalf("unexpected invalid response error: %v", err)
	}
}

func TestDispatchDialogMessageRoutesMatchingResponse(t *testing.T) {
	var gotID string
	var gotValue any
	handler := DialogHandler{
		MenuID:   "account",
		DialogID: "choice",
		OnResponse: func(dialogID string, value any) error {
			gotID = dialogID
			gotValue = value
			return nil
		},
	}

	handled, err := DispatchDialogMessage(
		[]byte("{\"type\":\"dialog.response\",\"menuId\":\"account\",\"menuGeneration\":7,\"dialogId\":\"choice\",\"value\":[\"a\",\"b\"]}"),
		handler,
	)
	if err != nil || !handled {
		t.Fatalf("dialog response handled = %v, err = %v", handled, err)
	}
	if gotID != "choice" || !reflect.DeepEqual(gotValue, []any{"a", "b"}) {
		t.Fatalf("callback got ID %q and value %#v", gotID, gotValue)
	}
	values, ok := DialogSelectionValuesFromAny(gotValue)
	if !ok || !reflect.DeepEqual(values, []string{"a", "b"}) {
		t.Fatalf("selection values = %#v, %v", values, ok)
	}

	handled, err = DispatchDialogMessage(DialogResponseMessage{
		MessageBase: MessageBase{Type: MessageDialogResponse, MenuID: "other"},
		DialogID:    "choice",
		Value:       nil,
	}, handler)
	if err != nil || handled {
		t.Fatalf("mismatched response handled = %v, err = %v", handled, err)
	}
}
