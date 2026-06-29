package formular

import (
	"encoding/json"
	"sort"
)

// MenuSnapshotState keeps the latest menu.snapshot-equivalent state for menus.
//
// It is intended for frontends and middleware that forward Formular messages and
// need to serve a newly connected frontend from locally cached menu state. It
// applies only messages that affect the observable menu snapshot. Ephemeral
// messages such as autocomplete hints and requests are ignored.
type MenuSnapshotState struct {
	menus map[string]MenuSnapshotMessage
}

// NewMenuSnapshotState creates an empty menu snapshot state cache.
func NewMenuSnapshotState() *MenuSnapshotState {
	return &MenuSnapshotState{menus: map[string]MenuSnapshotMessage{}}
}

// ApplyJSON decodes and applies one Formular message.
func (s *MenuSnapshotState) ApplyJSON(data []byte) (bool, error) {
	var base MessageBase
	if err := json.Unmarshal(data, &base); err != nil {
		return false, err
	}
	switch base.Type {
	case MessageMenuSnapshot:
		var msg MenuSnapshotMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return false, err
		}
		return s.Apply(msg), nil
	case MessageBlockSnapshot:
		var msg BlockSnapshotMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return false, err
		}
		return s.Apply(msg), nil
	case MessageBlockDelete:
		var msg BlockDeleteMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return false, err
		}
		return s.Apply(msg), nil
	case MessageFieldStatus:
		var msg FieldStatusMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return false, err
		}
		return s.Apply(msg), nil
	case MessageFieldUpdate:
		var msg FieldUpdateMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return false, err
		}
		return s.Apply(msg), nil
	case MessageFieldValidate:
		var msg FieldValidateMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return false, err
		}
		return s.Apply(msg), nil
	case MessageFormApply:
		var msg FormApplyMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			return false, err
		}
		return s.Apply(msg), nil
	default:
		return false, nil
	}
}

// Apply applies one typed Formular message or map-shaped decoded JSON message.
func (s *MenuSnapshotState) Apply(message any) bool {
	if s == nil {
		return false
	}
	if s.menus == nil {
		s.menus = map[string]MenuSnapshotMessage{}
	}
	switch msg := message.(type) {
	case MenuSnapshotMessage:
		return s.applyMenuSnapshot(msg)
	case *MenuSnapshotMessage:
		if msg == nil {
			return false
		}
		return s.applyMenuSnapshot(*msg)
	case BlockSnapshotMessage:
		return s.applyBlockSnapshot(msg)
	case *BlockSnapshotMessage:
		if msg == nil {
			return false
		}
		return s.applyBlockSnapshot(*msg)
	case BlockDeleteMessage:
		return s.applyBlockDelete(msg)
	case *BlockDeleteMessage:
		if msg == nil {
			return false
		}
		return s.applyBlockDelete(*msg)
	case FieldStatusMessage:
		return s.applyFieldStatus(msg)
	case *FieldStatusMessage:
		if msg == nil {
			return false
		}
		return s.applyFieldStatus(*msg)
	case FieldUpdateMessage:
		return s.applyFieldValue(msg.MenuID, msg.Field, msg.Value)
	case *FieldUpdateMessage:
		if msg == nil {
			return false
		}
		return s.applyFieldValue(msg.MenuID, msg.Field, msg.Value)
	case FieldValidateMessage:
		return s.applyFieldValidate(msg)
	case *FieldValidateMessage:
		if msg == nil {
			return false
		}
		return s.applyFieldValidate(*msg)
	case FormApplyMessage:
		return s.applyFormApply(msg)
	case *FormApplyMessage:
		if msg == nil {
			return false
		}
		return s.applyFormApply(*msg)
	case json.RawMessage:
		applied, _ := s.ApplyJSON(msg)
		return applied
	case []byte:
		applied, _ := s.ApplyJSON(msg)
		return applied
	case map[string]any:
		data, err := json.Marshal(msg)
		if err != nil {
			return false
		}
		applied, _ := s.ApplyJSON(data)
		return applied
	default:
		return false
	}
}

// Snapshot returns the cached snapshot for one menu.
func (s *MenuSnapshotState) Snapshot(menuID string) (MenuSnapshotMessage, bool) {
	if s == nil || s.menus == nil {
		return MenuSnapshotMessage{}, false
	}
	snapshot, ok := s.menus[menuID]
	if !ok {
		return MenuSnapshotMessage{}, false
	}
	return snapshot.Copy(), true
}

// ForceSnapshot returns the cached snapshot with Force set for frontend reloads.
func (s *MenuSnapshotState) ForceSnapshot(menuID string) (MenuSnapshotMessage, bool) {
	snapshot, ok := s.Snapshot(menuID)
	if !ok {
		return MenuSnapshotMessage{}, false
	}
	snapshot.Force = true
	return snapshot, true
}

// Snapshots returns all cached snapshots sorted by menu id.
func (s *MenuSnapshotState) Snapshots() []MenuSnapshotMessage {
	if s == nil || s.menus == nil {
		return nil
	}
	menuIDs := make([]string, 0, len(s.menus))
	for menuID := range s.menus {
		menuIDs = append(menuIDs, menuID)
	}
	sort.Strings(menuIDs)
	out := make([]MenuSnapshotMessage, 0, len(menuIDs))
	for _, menuID := range menuIDs {
		out = append(out, s.menus[menuID].Copy())
	}
	return out
}

// ForceSnapshots returns all cached snapshots sorted by menu id with Force set.
func (s *MenuSnapshotState) ForceSnapshots() []MenuSnapshotMessage {
	out := s.Snapshots()
	for i := range out {
		out[i].Force = true
	}
	return out
}

func (s *MenuSnapshotState) applyMenuSnapshot(msg MenuSnapshotMessage) bool {
	if msg.MenuID == "" {
		return false
	}
	s.menus[msg.MenuID] = msg.Copy()
	return true
}

func (s *MenuSnapshotState) applyBlockSnapshot(msg BlockSnapshotMessage) bool {
	if msg.MenuID == "" || msg.Block.ID == "" {
		return false
	}
	snapshot := s.ensureMenu(msg.MenuID, msg.MenuGeneration)
	snapshot.MenuGeneration = generationOr(snapshot.MenuGeneration, msg.MenuGeneration)
	replaced := false
	for i := range snapshot.Blocks {
		if snapshot.Blocks[i].ID == msg.Block.ID {
			snapshot.Blocks[i] = msg.Block.Copy()
			replaced = true
			break
		}
	}
	if !replaced {
		snapshot.Blocks = append(snapshot.Blocks, msg.Block.Copy())
	}
	s.menus[msg.MenuID] = snapshot
	return true
}

func (s *MenuSnapshotState) applyBlockDelete(msg BlockDeleteMessage) bool {
	if msg.MenuID == "" || msg.BlockID == "" {
		return false
	}
	snapshot, ok := s.menus[msg.MenuID]
	if !ok {
		return false
	}
	next := snapshot.Blocks[:0]
	changed := false
	for _, block := range snapshot.Blocks {
		if block.ID == msg.BlockID {
			changed = true
			continue
		}
		next = append(next, block)
	}
	if !changed {
		return false
	}
	snapshot.Blocks = copyBlocks(next)
	snapshot.MenuGeneration = generationOr(snapshot.MenuGeneration, msg.MenuGeneration)
	s.menus[msg.MenuID] = snapshot
	return true
}

func (s *MenuSnapshotState) applyFieldStatus(msg FieldStatusMessage) bool {
	return s.updateField(msg.MenuID, msg.Field, func(field *Field) {
		field.Status = msg.Status
		field.StatusText = msg.StatusText
		if msg.Readonly != nil {
			field.Readonly = *msg.Readonly
		}
	})
}

func (s *MenuSnapshotState) applyFieldValue(menuID string, ref FieldRef, value any) bool {
	return s.updateField(menuID, ref, func(field *Field) {
		field.Value = copyAny(value)
		if field.Kind == FieldArray {
			field.Elements = elementsFromValue(field, value)
		}
	})
}

func (s *MenuSnapshotState) applyFieldValidate(msg FieldValidateMessage) bool {
	return s.updateField(msg.MenuID, msg.Field, func(field *Field) {
		field.Value = copyAny(msg.Value)
		if field.Kind == FieldArray {
			field.Elements = elementsFromValue(field, msg.Value)
		}
		if field.Validation && field.Kind != FieldFile {
			field.Status = StatusUnset
			field.StatusText = ""
		}
	})
}

func (s *MenuSnapshotState) applyFormApply(msg FormApplyMessage) bool {
	snapshot, ok := s.menus[msg.MenuID]
	if !ok {
		return false
	}
	var block *Block
	for i := range snapshot.Blocks {
		if snapshot.Blocks[i].ID == msg.BlockID {
			block = &snapshot.Blocks[i]
			break
		}
	}
	if block == nil {
		return false
	}
	changed := applyValuesToItems(block.Items, msg.Values)
	if changed {
		s.menus[msg.MenuID] = snapshot
	}
	return changed
}

func (s *MenuSnapshotState) ensureMenu(menuID string, generation uint64) MenuSnapshotMessage {
	if snapshot, ok := s.menus[menuID]; ok {
		return snapshot.Copy()
	}
	return MenuSnapshotMessage{
		MessageBase: MessageBase{Type: MessageMenuSnapshot, MenuID: menuID, MenuGeneration: generation},
	}
}

func (s *MenuSnapshotState) updateField(menuID string, ref FieldRef, update func(*Field)) bool {
	snapshot, ok := s.menus[menuID]
	if !ok {
		return false
	}
	for blockIndex := range snapshot.Blocks {
		if snapshot.Blocks[blockIndex].ID != ref.BlockID {
			continue
		}
		field := findFieldInItems(snapshot.Blocks[blockIndex].Items, ref)
		if field == nil {
			return false
		}
		update(field)
		s.menus[menuID] = snapshot
		return true
	}
	return false
}

func findFieldInItems(items []Item, ref FieldRef) *Field {
	current := items
	for _, segment := range ref.ElementPath {
		var array *Field
		for i := range current {
			if current[i].Type == ItemField && current[i].ID == segment.ArrayFieldID {
				array = current[i].Field
				break
			}
		}
		if array == nil {
			return nil
		}
		var element *ArrayElement
		for i := range array.Elements {
			if array.Elements[i].ID == segment.ElementID {
				element = &array.Elements[i]
				break
			}
		}
		if element == nil {
			return nil
		}
		current = element.Items
	}
	for i := range current {
		if current[i].Type == ItemField && current[i].ID == ref.FieldID {
			return current[i].Field
		}
	}
	return nil
}

func applyValuesToItems(items []Item, values map[string]any) bool {
	changed := false
	for i := range items {
		if items[i].Type != ItemField || items[i].Field == nil {
			continue
		}
		value, ok := values[items[i].ID]
		if !ok {
			continue
		}
		items[i].Field.Value = copyAny(value) // nolint
		if items[i].Kind == FieldArray {
			items[i].Field.Elements = elementsFromValue(items[i].Field, value) // nolint
		}
		changed = true
	}
	return changed
}

func elementsFromValue(field *Field, value any) []ArrayElement {
	values, ok := ArrayElementValuesFromAny(value)
	if !ok || len(values) == 0 {
		return nil
	}
	out := make([]ArrayElement, 0, len(values))
	for _, item := range values {
		template := arrayTemplate(field, item.Template)
		element := ArrayElement{
			ID:       item.ID,
			Template: item.Template,
			Items:    copyItems(template.Items),
		}
		applyValuesToItems(element.Items, item.Values)
		out = append(out, element)
	}
	return out
}

func arrayTemplate(field *Field, name string) ArrayTemplate {
	for _, template := range field.Templates {
		if template.Name == name {
			return template
		}
	}
	return ArrayTemplate{Name: name}
}

func generationOr(current, next uint64) uint64 {
	if next != 0 {
		return next
	}
	return current
}
