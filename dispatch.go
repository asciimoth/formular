package formular

import (
	"encoding/json"
	"sort"
)

// Message is any decoded Formular protocol message.
//
// Dispatch helpers accept typed message structs, pointers to those structs,
// JSON bytes, json.RawMessage, and map-shaped decoded JSON values.
type Message = any

// FormHandler contains optional callbacks for frontend form messages routed to
// one menu and block.
type FormHandler struct {
	// MenuID filters messages by menuId when non-empty.
	MenuID string
	// BlockID filters messages by blockId when non-empty.
	BlockID string

	// OnApply receives each field value from a matching form.apply message.
	OnApply func(fieldID string, value any) error
	// OnValidate receives the field value from a matching field.validate message.
	OnValidate func(fieldID string, value any) error
	// OnUpdate receives the field value from a matching field.update message.
	OnUpdate func(fieldID string, value any) error
}

// DispatchFormMessage routes a frontend form message to a FormHandler.
//
// It returns handled=false for messages with a different menu or block, for
// unsupported message types, and for malformed JSON-like inputs. Matching
// messages are handled even when the corresponding callback is nil.
func DispatchFormMessage(msg Message, h FormHandler) (handled bool, err error) {
	switch typed := msg.(type) {
	case FieldUpdateMessage:
		return dispatchFieldValue(typed.MenuID, typed.Field, typed.Value, h, h.OnUpdate)
	case *FieldUpdateMessage:
		if typed == nil {
			return false, nil
		}
		return dispatchFieldValue(typed.MenuID, typed.Field, typed.Value, h, h.OnUpdate)
	case FieldValidateMessage:
		return dispatchFieldValue(typed.MenuID, typed.Field, typed.Value, h, h.OnValidate)
	case *FieldValidateMessage:
		if typed == nil {
			return false, nil
		}
		return dispatchFieldValue(typed.MenuID, typed.Field, typed.Value, h, h.OnValidate)
	case FormApplyMessage:
		return dispatchFormApply(typed, h)
	case *FormApplyMessage:
		if typed == nil {
			return false, nil
		}
		return dispatchFormApply(*typed, h)
	case json.RawMessage:
		return dispatchFormMessageJSON([]byte(typed), h)
	case []byte:
		return dispatchFormMessageJSON(typed, h)
	case map[string]any:
		data, marshalErr := json.Marshal(typed)
		if marshalErr != nil {
			return false, marshalErr
		}
		return dispatchFormMessageJSON(data, h)
	default:
		return false, nil
	}
}

func dispatchFormMessageJSON(data []byte, h FormHandler) (bool, error) {
	var base MessageBase
	if err := json.Unmarshal(data, &base); err != nil {
		return false, err
	}
	switch base.Type {
	case MessageFieldUpdate:
		var msg FieldUpdateMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return false, err
		}
		return DispatchFormMessage(msg, h)
	case MessageFieldValidate:
		var msg FieldValidateMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return false, err
		}
		return DispatchFormMessage(msg, h)
	case MessageFormApply:
		var msg FormApplyMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return false, err
		}
		return DispatchFormMessage(msg, h)
	default:
		return false, nil
	}
}

func dispatchFieldValue(menuID string, ref FieldRef, value any, h FormHandler, callback func(string, any) error) (bool, error) {
	if !handlerMatches(h, menuID, ref.BlockID) {
		return false, nil
	}
	if callback == nil {
		return true, nil
	}
	return true, callback(ref.FieldID, copyAny(value))
}

func dispatchFormApply(msg FormApplyMessage, h FormHandler) (bool, error) {
	if !handlerMatches(h, msg.MenuID, msg.BlockID) {
		return false, nil
	}
	if h.OnApply == nil {
		return true, nil
	}
	fieldIDs := make([]string, 0, len(msg.Values))
	for fieldID := range msg.Values {
		fieldIDs = append(fieldIDs, fieldID)
	}
	sort.Strings(fieldIDs)
	for _, fieldID := range fieldIDs {
		if err := h.OnApply(fieldID, copyAny(msg.Values[fieldID])); err != nil {
			return true, err
		}
	}
	return true, nil
}

func handlerMatches(h FormHandler, menuID, blockID string) bool {
	if h.MenuID != "" && h.MenuID != menuID {
		return false
	}
	if h.BlockID != "" && h.BlockID != blockID {
		return false
	}
	return true
}
