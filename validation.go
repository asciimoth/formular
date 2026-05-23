package formular

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
)

// ValidationError describes one invalid field in a Formular structure.
type ValidationError struct {
	// Path identifies the invalid field or value.
	Path string
	// Message describes the validation failure.
	Message string
}

// Error returns a human-readable validation error.
func (e ValidationError) Error() string {
	if e.Path == "" {
		return e.Message
	}
	return e.Path + ": " + e.Message
}

// Copy returns a deep copy of e.
func (e ValidationError) Copy() ValidationError {
	return e
}

// Validate returns an error if e is not a valid validation error.
func (e ValidationError) Validate() error {
	if e.Message == "" {
		return ValidationErrors{{Path: "validationError.message", Message: "must not be empty"}}
	}
	return nil
}

// ValidationErrors is a collection of validation failures.
type ValidationErrors []ValidationError

// Error returns all validation failures as one string.
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	out := e[0].Error()
	for _, err := range e[1:] {
		out += "; " + err.Error()
	}
	return out
}

// Unwrap returns the validation failures for errors.Join-style inspection.
func (e ValidationErrors) Unwrap() []error {
	out := make([]error, len(e))
	for i := range e {
		out[i] = e[i]
	}
	return out
}

type validator struct {
	errs ValidationErrors
}

// Validate returns an error if m is not a valid message base.
func (m MessageBase) Validate() error {
	var v validator
	v.messageBase("message", m, "")
	return v.err()
}

// Validate returns an error if m is not a valid menu snapshot message.
func (m MenuSnapshotMessage) Validate() error {
	var v validator
	v.messageBase("message", m.MessageBase, MessageMenuSnapshot)
	v.blocks("message.blocks", m.Blocks)
	return v.err()
}

// Validate returns an error if m is not a valid block snapshot message.
func (m BlockSnapshotMessage) Validate() error {
	var v validator
	v.messageBase("message", m.MessageBase, MessageBlockSnapshot)
	v.block("message.block", m.Block)
	return v.err()
}

// Validate returns an error if m is not a valid block delete message.
func (m BlockDeleteMessage) Validate() error {
	var v validator
	v.messageBase("message", m.MessageBase, MessageBlockDelete)
	v.requiredID("message.blockId", m.BlockID)
	return v.err()
}

// Validate returns an error if m is not a valid field status message.
func (m FieldStatusMessage) Validate() error {
	var v validator
	v.messageBase("message", m.MessageBase, MessageFieldStatus)
	v.fieldRef("message.field", m.Field)
	v.validationStatus("message.status", m.Status, true)
	return v.err()
}

// Validate returns an error if m is not a valid autocomplete hints message.
func (m AutocompleteHintsMessage) Validate() error {
	var v validator
	v.messageBase("message", m.MessageBase, MessageAutocompleteHints)
	v.fieldRef("message.field", m.Field)
	if m.Hints == nil {
		v.add("message.hints", "must not be nil")
	}
	return v.err()
}

// Validate returns an error if m is not a valid field update message.
func (m FieldUpdateMessage) Validate() error {
	var v validator
	v.messageBase("message", m.MessageBase, MessageFieldUpdate)
	v.fieldRef("message.field", m.Field)
	v.value("message.value", m.Value)
	return v.err()
}

// Validate returns an error if m is not a valid field validate message.
func (m FieldValidateMessage) Validate() error {
	var v validator
	v.messageBase("message", m.MessageBase, MessageFieldValidate)
	v.fieldRef("message.field", m.Field)
	v.value("message.value", m.Value)
	return v.err()
}

// Validate returns an error if m is not a valid form apply message.
func (m FormApplyMessage) Validate() error {
	var v validator
	v.messageBase("message", m.MessageBase, MessageFormApply)
	v.requiredID("message.blockId", m.BlockID)
	if m.Values == nil {
		v.add("message.values", "must not be nil")
	}
	v.valueMap("message.values", m.Values)
	return v.err()
}

// Validate returns an error if m is not a valid button press message.
func (m ButtonPressMessage) Validate() error {
	var v validator
	v.messageBase("message", m.MessageBase, MessageButtonPress)
	v.requiredID("message.blockId", m.BlockID)
	v.elementPath("message.elementPath", m.ElementPath)
	v.requiredID("message.buttonId", m.ButtonID)
	return v.err()
}

// Validate returns an error if m is not a valid autocomplete request message.
func (m AutocompleteRequestMessage) Validate() error {
	var v validator
	v.messageBase("message", m.MessageBase, MessageAutocompleteRequest)
	v.fieldRef("message.field", m.Field)
	return v.err()
}

// Validate returns an error if b is not a valid block.
func (b Block) Validate() error {
	var v validator
	v.block("block", b)
	return v.err()
}

// Validate returns an error if i is not a valid item.
func (i Item) Validate() error {
	var v validator
	v.item("item", i, false)
	return v.err()
}

// Validate returns an error if l is not a valid log line.
func (l LogLine) Validate() error {
	var v validator
	v.logLine("logLine", l)
	return v.err()
}

// Validate returns an error if f is not a valid field.
func (f Field) Validate() error {
	var v validator
	v.field("field", f, false)
	return v.err()
}

// Validate returns an error if a is not a valid autocomplete configuration.
func (a Autocomplete) Validate() error {
	return nil
}

// Validate returns an error if t is not a valid array template.
func (t ArrayTemplate) Validate() error {
	var v validator
	v.arrayTemplate("arrayTemplate", t)
	return v.err()
}

// Validate returns an error if e is not a valid array element.
func (e ArrayElement) Validate() error {
	var v validator
	v.arrayElement("arrayElement", e, nil)
	return v.err()
}

// Validate returns an error if v is not a valid array element value.
func (v ArrayElementValue) Validate() error {
	var val validator
	val.arrayElementValue("arrayElementValue", v)
	return val.err()
}

// Validate returns an error if r is not a valid field reference.
func (r FieldRef) Validate() error {
	var v validator
	v.fieldRef("fieldRef", r)
	return v.err()
}

// Validate returns an error if s is not a valid element path segment.
func (s ElementPathSegment) Validate() error {
	var v validator
	v.elementPathSegment("elementPathSegment", s)
	return v.err()
}

// Validate returns an error if v is not a valid field value.
func (v FieldValue) Validate() error {
	var val validator
	val.requiredID("fieldValue.fieldId", v.FieldID)
	val.value("fieldValue.value", v.Value)
	return val.err()
}

// Validate returns an error if c is not a valid copyable descriptor.
func (c Copyable) Validate() error {
	return nil
}

func (v *validator) err() error {
	if len(v.errs) == 0 {
		return nil
	}
	return v.errs
}

func (v *validator) add(path, message string) {
	v.errs = append(v.errs, ValidationError{Path: path, Message: message})
}

func (v *validator) messageBase(path string, m MessageBase, wantType string) {
	if wantType != "" && m.Type != wantType {
		v.add(path+".type", fmt.Sprintf("must be %q", wantType))
	} else if wantType == "" {
		v.requiredID(path+".type", m.Type)
	}
	v.requiredID(path+".menuId", m.MenuID)
}

func (v *validator) blocks(path string, blocks []Block) {
	if blocks == nil {
		v.add(path, "must not be nil")
		return
	}
	ids := map[string]int{}
	for i, block := range blocks {
		itemPath := indexPath(path, i)
		if prev, ok := ids[block.ID]; ok && block.ID != "" {
			v.add(itemPath+".id", fmt.Sprintf("duplicates %s.id", indexPath(path, prev)))
		}
		ids[block.ID] = i
		v.block(itemPath, block)
	}
}

func (v *validator) block(path string, block Block) {
	v.requiredID(path+".id", block.ID)
	if block.Copyable != nil {
		v.copyable(path+".copyable", *block.Copyable)
	}
	if block.Items == nil {
		v.add(path+".items", "must not be nil")
		return
	}
	v.items(path+".items", block.Items, false)
}

func (v *validator) items(path string, items []Item, inArrayElement bool) {
	ids := map[string]int{}
	for i, item := range items {
		itemPath := indexPath(path, i)
		if prev, ok := ids[item.ID]; ok && item.ID != "" {
			v.add(itemPath+".id", fmt.Sprintf("duplicates %s.id", indexPath(path, prev)))
		}
		ids[item.ID] = i
		v.item(itemPath, item, inArrayElement)
	}
}

func (v *validator) item(path string, item Item, inArrayElement bool) {
	v.requiredID(path+".type", item.Type)
	v.requiredID(path+".id", item.ID)
	switch item.Type {
	case ItemHeader:
		if inArrayElement {
			v.add(path+".type", "headers are not allowed in array elements")
		}
		if item.Text == "" {
			v.add(path+".text", "must not be empty for header items")
		}
		if item.Field != nil {
			v.add(path+".field", "must be nil for header items")
		}
	case ItemLabel:
		if item.Text == "" {
			v.add(path+".text", "must not be empty for label items")
		}
		v.textFormat(path+".format", item.Format, false)
		if item.Format != TextCode && item.Syntax != "" {
			v.add(path+".syntax", "requires format code")
		}
		if item.Field != nil {
			v.add(path+".field", "must be nil for label items")
		}
	case ItemProgressbar:
		if item.Label == "" {
			v.add(path+".label", "must not be empty for progressbar items")
		}
		if item.Progress == nil {
			v.add(path+".progress", "must not be nil for progressbar items")
		} else if *item.Progress > 100 {
			v.add(path+".progress", "must be between 0 and 100")
		}
		if item.Field != nil {
			v.add(path+".field", "must be nil for progressbar items")
		}
	case ItemLogs:
		if item.Label == "" {
			v.add(path+".label", "must not be empty for logs items")
		}
		if item.Logs == nil {
			v.add(path+".logs", "must not be nil for logs items")
		}
		for i, line := range item.Logs {
			v.logLine(indexPath(path+".logs", i), line)
		}
		if item.Field != nil {
			v.add(path+".field", "must be nil for logs items")
		}
	case ItemButton:
		if item.Label == "" {
			v.add(path+".label", "must not be empty for button items")
		}
		if item.Field != nil {
			v.add(path+".field", "must be nil for button items")
		}
	case ItemField:
		if item.Label == "" {
			v.add(path+".label", "must not be empty for field items")
		}
		if item.Field == nil {
			v.add(path+".field", "must not be nil for field items")
			return
		}
		v.field(path, *item.Field, inArrayElement)
	default:
		v.add(path+".type", "must be one of header, label, progressbar, logs, button, field")
	}
}

func (v *validator) logLine(path string, line LogLine) {
	v.logLevel(path+".level", line.Level)
	if line.Text == "" {
		v.add(path+".text", "must not be empty")
	}
}

func (v *validator) field(path string, field Field, inArrayElement bool) {
	v.fieldKind(path+".kind", field.Kind)
	v.validationStatus(path+".status", field.Status, false)
	if field.Value != nil {
		v.value(path+".value", field.Value)
		v.valueForKind(path+".value", field.Kind, field.Value)
	}
	for i, value := range field.AllowedValues {
		valuePath := indexPath(path+".allowedValues", i)
		v.value(valuePath, value)
		v.allowedValueForKind(valuePath, field.Kind, value)
	}
	if field.Min != nil && math.IsNaN(*field.Min) {
		v.add(path+".min", "must not be NaN")
	}
	if field.Max != nil && math.IsNaN(*field.Max) {
		v.add(path+".max", "must not be NaN")
	}
	if field.Min != nil && field.Max != nil && *field.Min > *field.Max {
		v.add(path+".min", "must be less than or equal to max")
	}
	if field.MaxBytes != nil && *field.MaxBytes == 0 {
		v.add(path+".maxBytes", "must be greater than zero")
	}
	for i, accept := range field.Accept {
		if accept == "" {
			v.add(indexPath(path+".accept", i), "must not be empty")
		}
	}

	switch field.Kind {
	case FieldText:
		if field.MaxBytes != nil {
			v.add(path+".maxBytes", "is only valid for file fields")
		}
		if len(field.Accept) > 0 {
			v.add(path+".accept", "is only valid for file fields")
		}
		if len(field.Templates) > 0 || len(field.Elements) > 0 {
			v.add(path+".templates", "is only valid for array fields")
		}
	case FieldInt:
		v.noTextOnlyOptions(path, field)
		v.noFileOnlyOptions(path, field)
		v.noArrayOnlyOptions(path, field)
	case FieldFloat:
		v.noTextOnlyOptions(path, field)
		v.noFileOnlyOptions(path, field)
		v.noArrayOnlyOptions(path, field)
	case FieldRange:
		v.noTextOnlyOptions(path, field)
		v.noFileOnlyOptions(path, field)
		v.noArrayOnlyOptions(path, field)
		if field.Fraction != nil {
			v.add(path+".fraction", "is only valid for float fields")
		}
		if len(field.AllowedValues) > 0 {
			v.add(path+".allowedValues", "is not valid for range fields")
		}
	case FieldFile:
		v.noTextOnlyOptions(path, field)
		v.noNumberOnlyOptions(path, field)
		v.noArrayOnlyOptions(path, field)
		if len(field.AllowedValues) > 0 {
			v.add(path+".allowedValues", "is not valid for file fields")
		}
		if field.Validation {
			v.add(path+".validate", "must be false for file fields")
		}
	case FieldCheckbox:
		v.noTextOnlyOptions(path, field)
		v.noNumberOnlyOptions(path, field)
		v.noFileOnlyOptions(path, field)
		v.noArrayOnlyOptions(path, field)
		if len(field.AllowedValues) > 0 {
			v.add(path+".allowedValues", "is not valid for checkbox fields")
		}
	case FieldRadio:
		v.noTextOnlyOptions(path, field)
		v.noNumberOnlyOptions(path, field)
		v.noFileOnlyOptions(path, field)
		v.noArrayOnlyOptions(path, field)
		if len(field.AllowedValues) == 0 {
			v.add(path+".allowedValues", "must contain at least one value for radio fields")
		}
		if field.Value != nil && len(field.AllowedValues) > 0 && !containsValue(field.AllowedValues, field.Value) {
			v.add(path+".value", "must match one of allowedValues for radio fields")
		}
	case FieldArray:
		v.noTextOnlyOptions(path, field)
		v.noNumberOnlyOptions(path, field)
		v.noFileOnlyOptions(path, field)
		if len(field.AllowedValues) > 0 {
			v.add(path+".allowedValues", "is not valid for array fields")
		}
		if inArrayElement {
			v.add(path+".kind", "array fields are not allowed in array elements")
		}
		v.arrayTemplates(path+".templates", field.Templates)
		v.arrayElements(path+".elements", field.Elements, field.Templates)
	default:
		// fieldKind already reported the invalid kind.
	}
}

func (v *validator) valueForKind(path, kind string, value any) {
	switch kind {
	case FieldText:
		if !isString(value) {
			v.add(path, "must be a string for text fields")
		}
	case FieldInt:
		if !isInteger(value) {
			v.add(path, "must be an integer for int fields")
		}
	case FieldFloat, FieldRange:
		if !isNumber(value) {
			v.add(path, "must be a number")
		}
	case FieldFile:
		s, ok := value.(string)
		if !ok {
			v.add(path, "must be a base64 string for file fields")
			return
		}
		if _, err := base64.StdEncoding.DecodeString(s); err != nil {
			v.add(path, "must be valid base64 for file fields")
		}
	case FieldCheckbox:
		if _, ok := value.(bool); !ok {
			v.add(path, "must be a boolean for checkbox fields")
		}
	case FieldRadio:
		// Radio values can use any scalar protocol value.
	case FieldArray:
		switch value.(type) {
		case []ArrayElementValue, []any:
		default:
			v.add(path, "must be an array of array element values")
		}
	}
}

func (v *validator) allowedValueForKind(path, kind string, value any) {
	switch kind {
	case FieldText:
		if !isString(value) {
			v.add(path, "must be a string")
		}
	case FieldInt:
		if !isInteger(value) {
			v.add(path, "must be an integer")
		}
	case FieldFloat:
		if !isNumber(value) {
			v.add(path, "must be a number")
		}
	case FieldRadio:
		// Radio choices can use any scalar protocol value.
	}
}

func (v *validator) noTextOnlyOptions(path string, field Field) {
	if field.Secret {
		v.add(path+".secret", "is only valid for text fields")
	}
	if field.Multiline {
		v.add(path+".multiline", "is only valid for text fields")
	}
	if field.Subtype != "" {
		v.add(path+".subtype", "is only valid for text fields")
	}
	if field.Autocomplete != nil {
		v.add(path+".autocomplete", "is only valid for text fields")
	}
}

func (v *validator) noNumberOnlyOptions(path string, field Field) {
	if field.Min != nil {
		v.add(path+".min", "is only valid for int, float, and range fields")
	}
	if field.Max != nil {
		v.add(path+".max", "is only valid for int, float, and range fields")
	}
	if field.Fraction != nil {
		v.add(path+".fraction", "is only valid for float fields")
	}
}

func (v *validator) noFileOnlyOptions(path string, field Field) {
	if field.MaxBytes != nil {
		v.add(path+".maxBytes", "is only valid for file fields")
	}
	if len(field.Accept) > 0 {
		v.add(path+".accept", "is only valid for file fields")
	}
}

func (v *validator) noArrayOnlyOptions(path string, field Field) {
	if len(field.Templates) > 0 {
		v.add(path+".templates", "is only valid for array fields")
	}
	if len(field.Elements) > 0 {
		v.add(path+".elements", "is only valid for array fields")
	}
}

func (v *validator) arrayTemplates(path string, templates []ArrayTemplate) {
	ids := map[string]int{}
	for i, tmpl := range templates {
		itemPath := indexPath(path, i)
		if prev, ok := ids[tmpl.Name]; ok && tmpl.Name != "" {
			v.add(itemPath+".name", fmt.Sprintf("duplicates %s.name", indexPath(path, prev)))
		}
		ids[tmpl.Name] = i
		v.arrayTemplate(itemPath, tmpl)
	}
}

func (v *validator) arrayTemplate(path string, tmpl ArrayTemplate) {
	v.requiredID(path+".name", tmpl.Name)
	if tmpl.Items == nil {
		v.add(path+".items", "must not be nil")
		return
	}
	v.items(path+".items", tmpl.Items, true)
}

func (v *validator) arrayElements(path string, elements []ArrayElement, templates []ArrayTemplate) {
	templateNames := map[string]struct{}{}
	for _, tmpl := range templates {
		if tmpl.Name != "" {
			templateNames[tmpl.Name] = struct{}{}
		}
	}
	ids := map[string]int{}
	for i, elem := range elements {
		itemPath := indexPath(path, i)
		if prev, ok := ids[elem.ID]; ok && elem.ID != "" {
			v.add(itemPath+".id", fmt.Sprintf("duplicates %s.id", indexPath(path, prev)))
		}
		ids[elem.ID] = i
		v.arrayElement(itemPath, elem, templateNames)
	}
}

func (v *validator) arrayElement(path string, elem ArrayElement, templateNames map[string]struct{}) {
	v.requiredID(path+".id", elem.ID)
	v.requiredID(path+".template", elem.Template)
	if templateNames != nil && elem.Template != "" {
		if _, ok := templateNames[elem.Template]; !ok {
			v.add(path+".template", "must match one of the array field templates")
		}
	}
	if elem.Items == nil {
		v.add(path+".items", "must not be nil")
		return
	}
	v.items(path+".items", elem.Items, true)
	if elem.Copyable != nil {
		v.copyable(path+".copyable", *elem.Copyable)
	}
}

func (v *validator) arrayElementValue(path string, value ArrayElementValue) {
	v.requiredID(path+".id", value.ID)
	v.requiredID(path+".template", value.Template)
	if value.Values == nil {
		v.add(path+".values", "must not be nil")
		return
	}
	v.valueMap(path+".values", value.Values)
}

func (v *validator) fieldRef(path string, ref FieldRef) {
	v.requiredID(path+".blockId", ref.BlockID)
	v.requiredID(path+".fieldId", ref.FieldID)
	v.elementPath(path+".elementPath", ref.ElementPath)
}

func (v *validator) elementPath(path string, segments []ElementPathSegment) {
	for i, segment := range segments {
		v.elementPathSegment(indexPath(path, i), segment)
	}
}

func (v *validator) elementPathSegment(path string, segment ElementPathSegment) {
	v.requiredID(path+".arrayFieldId", segment.ArrayFieldID)
	v.requiredID(path+".elementId", segment.ElementID)
}

func (v *validator) copyable(path string, copyable Copyable) {
	_ = path
	_ = copyable
}

func (v *validator) valueMap(path string, values map[string]any) {
	for key, value := range values {
		if key == "" {
			v.add(path, "contains empty key")
			continue
		}
		v.value(path+"."+key, value)
	}
}

func (v *validator) value(path string, value any) {
	switch typed := value.(type) {
	case nil:
	case string, bool:
	case int, int8, int16, int32, int64:
	case uint, uint8, uint16, uint32, uint64:
	case json.Number:
		if _, err := typed.Float64(); err != nil {
			v.add(path, "must be a valid number")
		}
	case float32:
		if math.IsNaN(float64(typed)) || math.IsInf(float64(typed), 0) {
			v.add(path, "must be a finite number")
		}
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			v.add(path, "must be a finite number")
		}
	case []ArrayElementValue:
		for i := range typed {
			v.arrayElementValue(indexPath(path, i), typed[i])
		}
	case []any:
		for i := range typed {
			v.arrayElementValueAny(indexPath(path, i), typed[i])
		}
	case map[string]any:
		v.add(path, "objects are not valid field values")
	case ArrayElementValue:
		v.arrayElementValue(path, typed)
	case FieldValue:
		v.requiredID(path+".fieldId", typed.FieldID)
		v.value(path+".value", typed.Value)
	default:
		v.reflectValue(path, reflect.ValueOf(value))
	}
}

func (v *validator) arrayElementValueAny(path string, value any) {
	switch typed := value.(type) {
	case ArrayElementValue:
		v.arrayElementValue(path, typed)
	case map[string]any:
		v.requiredID(path+".id", stringFromAny(typed["id"]))
		v.requiredID(path+".template", stringFromAny(typed["template"]))
		rawValues, ok := typed["values"].(map[string]any)
		if !ok {
			v.add(path+".values", "must be an object")
			return
		}
		v.valueMap(path+".values", rawValues)
	default:
		v.add(path, "array values must contain array element values")
	}
}

func (v *validator) reflectValue(path string, value reflect.Value) {
	if !value.IsValid() {
		return
	}
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.String, reflect.Bool:
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	case reflect.Float32, reflect.Float64:
		f := value.Convert(reflect.TypeOf(float64(0))).Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			v.add(path, "must be a finite number")
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			v.value(indexPath(path, i), value.Index(i).Interface())
		}
	case reflect.Map:
		v.add(path, "objects are not valid field values")
	default:
		v.add(path, fmt.Sprintf("unsupported value type %T", value.Interface()))
	}
}

func (v *validator) requiredID(path, value string) {
	if value == "" {
		v.add(path, "must not be empty")
	}
}

func (v *validator) textFormat(path, value string, required bool) {
	if value == "" && !required {
		return
	}
	switch value {
	case TextPlain, TextMarkdown, TextCode:
	default:
		v.add(path, "must be one of plain, markdown, code")
	}
}

func (v *validator) validationStatus(path, value string, required bool) {
	if value == "" && !required {
		return
	}
	switch value {
	case StatusUnset, StatusOK, StatusWarn, StatusError:
	default:
		v.add(path, "must be one of unset, ok, warn, error")
	}
}

func (v *validator) logLevel(path, value string) {
	switch value {
	case LogTrace, LogDebug, LogInfo, LogWarn, LogError, LogPanic:
	default:
		v.add(path, "must be one of trace, debug, info, warn, error, panic")
	}
}

func (v *validator) fieldKind(path, value string) {
	switch value {
	case FieldText, FieldInt, FieldFloat, FieldFile, FieldCheckbox, FieldRadio, FieldRange, FieldArray:
	default:
		v.add(path, "must be one of text, int, float, file, checkbox, radio, range, array")
	}
}

func indexPath(path string, i int) string {
	return fmt.Sprintf("%s[%d]", path, i)
}

func stringFromAny(value any) string {
	s, _ := value.(string)
	return s
}

func isString(value any) bool {
	_, ok := value.(string)
	return ok
}

func isInteger(value any) bool {
	switch typed := value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case json.Number:
		f, err := typed.Float64()
		return err == nil && isFinite(f) && math.Trunc(f) == f
	case float32:
		f := float64(typed)
		return isFinite(f) && math.Trunc(f) == f
	case float64:
		return isFinite(typed) && math.Trunc(typed) == typed
	default:
		return false
	}
}

func isNumber(value any) bool {
	switch typed := value.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case json.Number:
		f, err := typed.Float64()
		return err == nil && isFinite(f)
	case float32:
		return isFinite(float64(typed))
	case float64:
		return isFinite(typed)
	default:
		return false
	}
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func containsValue(values []any, value any) bool {
	for _, candidate := range values {
		if reflect.DeepEqual(candidate, value) {
			return true
		}
	}
	return false
}

// IsValidationError reports whether err contains one or more Formular
// validation errors.
func IsValidationError(err error) bool {
	var errs ValidationErrors
	return errors.As(err, &errs)
}
