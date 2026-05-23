// nolint
package formular_test

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	formular "github.com/asciimoth/formular"
)

type routedMessage struct {
	client string
	msg    any
}

type testBackend struct {
	inbox   chan routedMessage
	outbox  chan routedMessage
	menus   map[string]formular.MenuSnapshotMessage
	handled []any
	ignored int
}

func newTestBackend(menus ...formular.MenuSnapshotMessage) *testBackend {
	b := &testBackend{
		inbox:  make(chan routedMessage, 64),
		outbox: make(chan routedMessage, 64),
		menus:  map[string]formular.MenuSnapshotMessage{},
	}
	for _, menu := range menus {
		b.menus[menu.MenuID] = menu.Copy()
	}
	return b
}

func (b *testBackend) sendSnapshot(client, menuID string) {
	b.outbox <- routedMessage{client: client, msg: b.menus[menuID].Copy()}
}

func (b *testBackend) send(client string, msg any) {
	b.outbox <- routedMessage{client: client, msg: msg}
}

func (b *testBackend) receiveOne(t *testing.T) {
	t.Helper()

	select {
	case envelope := <-b.inbox:
		if !b.handle(envelope.msg) {
			b.ignored++
		}
	default:
		t.Fatal("backend inbox was empty")
	}
}

func (b *testBackend) handle(msg any) bool {
	if err := validateFrontendMessage(msg); err != nil {
		return false
	}
	base := frontendMessageBase(msg)
	menu, ok := b.menus[base.MenuID]
	if !ok || base.MenuGeneration != menu.MenuGeneration {
		return false
	}
	if blockID, ok := frontendMessageBlockID(msg); ok {
		block, found := findBlock(menu, blockID)
		if !found || base.BlockGeneration != block.Generation {
			return false
		}
	}
	b.handled = append(b.handled, msg)
	return true
}

type testFrontend struct {
	name       string
	inbox      chan any
	outbox     chan any
	menus      map[string]*menuView
	activeMenu string
	focused    *focusedField
	hints      []string
	ignored    int
}

type focusedField struct {
	menuID string
	field  formular.FieldRef
	prefix string
}

type menuView struct {
	generation uint64
	blocks     map[string]*blockView
	order      []string
}

type blockView struct {
	block     formular.Block
	fields    map[fieldKey]*fieldView
	collapsed bool
}

type fieldView struct {
	item       *formular.Item
	value      any
	status     string
	statusText string
	readonly   bool
}

type fieldKey struct {
	blockID string
	path    string
	fieldID string
}

func newTestFrontend(name string) *testFrontend {
	return &testFrontend{
		name:   name,
		inbox:  make(chan any, 64),
		outbox: make(chan any, 64),
		menus:  map[string]*menuView{},
	}
}

func (f *testFrontend) receiveOne(t *testing.T) {
	t.Helper()

	select {
	case msg := <-f.inbox:
		if !f.handle(msg) {
			f.ignored++
		}
	default:
		t.Fatalf("%s inbox was empty", f.name)
	}
}

func (f *testFrontend) handle(msg any) bool {
	switch m := msg.(type) {
	case formular.MenuSnapshotMessage:
		if err := m.Validate(); err != nil {
			return false
		}
		if current := f.menus[m.MenuID]; current != nil && !m.Force {
			for i := range m.Blocks {
				if old := current.blocks[m.Blocks[i].ID]; old != nil {
					m.Blocks[i].Collapsed = old.collapsed
				}
			}
		}
		f.menus[m.MenuID] = newMenuView(m)
		if f.activeMenu == "" {
			f.activeMenu = m.MenuID
		}
		return true
	case formular.BlockSnapshotMessage:
		if err := m.Validate(); err != nil {
			return false
		}
		menu := f.menus[m.MenuID]
		if menu == nil {
			return false
		}
		menu.putBlock(m.Block)
		return true
	case formular.BlockDeleteMessage:
		if err := m.Validate(); err != nil {
			return false
		}
		menu := f.menus[m.MenuID]
		if menu == nil {
			return false
		}
		delete(menu.blocks, m.BlockID)
		menu.rebuildOrder()
		return true
	case formular.FieldStatusMessage:
		if err := m.Validate(); err != nil {
			return false
		}
		field := f.field(m.MenuID, m.Field)
		if field == nil {
			return false
		}
		block := f.menus[m.MenuID].blocks[m.Field.BlockID]
		if m.BlockGeneration < block.block.Generation {
			return false
		}
		field.status = m.Status
		field.statusText = m.StatusText
		if m.Readonly != nil {
			field.readonly = *m.Readonly
		}
		return true
	case formular.AutocompleteHintsMessage:
		if err := m.Validate(); err != nil {
			return false
		}
		if f.focused == nil || f.focused.menuID != m.MenuID || !sameFieldRef(f.focused.field, m.Field) {
			return false
		}
		for _, hint := range m.Hints {
			if !strings.HasPrefix(hint, f.focused.prefix) {
				return false
			}
		}
		f.hints = append([]string(nil), m.Hints...)
		return true
	default:
		return false
	}
}

func (f *testFrontend) collapse(menuID, blockID string, collapsed bool) {
	f.menus[menuID].blocks[blockID].collapsed = collapsed
}

func (f *testFrontend) focus(menuID string, ref formular.FieldRef, prefix string) {
	f.focused = &focusedField{menuID: menuID, field: ref.Copy(), prefix: prefix}
}

func (f *testFrontend) updateField(menuID string, ref formular.FieldRef, value any) {
	field := f.field(menuID, ref)
	if field == nil || field.readonly {
		return
	}
	block := f.menus[menuID].blocks[ref.BlockID]
	if block.block.Inactive {
		return
	}
	field.value = value
	if block.block.Form {
		if field.item.Field.Validation && field.item.Field.Kind != formular.FieldFile {
			f.outbox <- formular.FieldValidateMessage{
				MessageBase: frontendBase(formular.MessageFieldValidate, menuID, f.menus[menuID].generation, block.block.Generation),
				Field:       ref.Copy(),
				Value:       copyValue(value),
			}
		}
		return
	}
	f.outbox <- formular.FieldUpdateMessage{
		MessageBase: frontendBase(formular.MessageFieldUpdate, menuID, f.menus[menuID].generation, block.block.Generation),
		Field:       ref.Copy(),
		Value:       copyValue(value),
	}
	if field.item.Field.Validation && field.item.Field.Kind != formular.FieldFile {
		f.outbox <- formular.FieldValidateMessage{
			MessageBase: frontendBase(formular.MessageFieldValidate, menuID, f.menus[menuID].generation, block.block.Generation),
			Field:       ref.Copy(),
			Value:       copyValue(value),
		}
	}
}

func (f *testFrontend) requestAutocomplete(menuID string, ref formular.FieldRef, prefix string) {
	field := f.field(menuID, ref)
	if field == nil || field.item.Field.Autocomplete == nil || !field.item.Field.Autocomplete.Enabled {
		return
	}
	block := f.menus[menuID].blocks[ref.BlockID]
	f.focus(menuID, ref, prefix)
	f.outbox <- formular.AutocompleteRequestMessage{
		MessageBase: frontendBase(formular.MessageAutocompleteRequest, menuID, f.menus[menuID].generation, block.block.Generation),
		Field:       ref.Copy(),
		Prefix:      prefix,
	}
}

func (f *testFrontend) pressButton(menuID, blockID, buttonID string, path []formular.ElementPathSegment) {
	block := f.menus[menuID].blocks[blockID]
	if block.block.Inactive || !buttonActive(block.block, buttonID, path) {
		return
	}
	f.outbox <- formular.ButtonPressMessage{
		MessageBase: frontendBase(formular.MessageButtonPress, menuID, f.menus[menuID].generation, block.block.Generation),
		BlockID:     blockID,
		ElementPath: append([]formular.ElementPathSegment(nil), path...),
		ButtonID:    buttonID,
	}
}

func (f *testFrontend) applyForm(menuID, blockID string) {
	menu := f.menus[menuID]
	block := menu.blocks[blockID]
	if !block.block.Form || block.block.Inactive {
		return
	}
	values := map[string]any{}
	for key, field := range block.fields {
		if len(field.item.Field.Templates) > 0 || len(key.path) == 0 {
			if field.item.Field.Required && emptyValue(field.value) {
				return
			}
			if field.item.Field.Validation && field.item.Field.Kind != formular.FieldFile && field.status != formular.StatusOK {
				return
			}
			values[key.fieldID] = copyValue(field.value)
		}
	}
	f.outbox <- formular.FormApplyMessage{
		MessageBase: frontendBase(formular.MessageFormApply, menuID, menu.generation, block.block.Generation),
		BlockID:     blockID,
		Values:      values,
	}
}

func (f *testFrontend) field(menuID string, ref formular.FieldRef) *fieldView {
	menu := f.menus[menuID]
	if menu == nil {
		return nil
	}
	block := menu.blocks[ref.BlockID]
	if block == nil {
		return nil
	}
	return block.fields[keyFromRef(ref)]
}

func TestE2EBackendWithTwoFrontendsHonorsProtocolContracts(t *testing.T) {
	settings := settingsMenu()
	tools := toolsMenu()
	for _, menu := range []formular.MenuSnapshotMessage{settings, tools} {
		if err := menu.Validate(); err != nil {
			t.Fatalf("fixture menu %q failed validation: %v", menu.MenuID, err)
		}
	}

	backend := newTestBackend(settings, tools)
	left := newTestFrontend("left")
	right := newTestFrontend("right")

	backend.sendSnapshot("left", "settings")
	backend.sendSnapshot("right", "settings")
	deliverBackend(t, backend, left, right, 2)

	assertBlocks(t, left, "settings", []string{"live", "profile", "upload", "disabled"})
	assertBlocks(t, right, "settings", []string{"live", "profile", "upload", "disabled"})

	left.collapse("settings", "live", true)
	backend.send("left", settings.Copy())
	deliverBackend(t, backend, left, right, 1)
	if !left.menus["settings"].blocks["live"].collapsed {
		t.Fatal("non-forced menu snapshot should preserve local collapse state")
	}
	forced := settings.Copy()
	forced.Force = true
	backend.send("left", forced)
	deliverBackend(t, backend, left, right, 1)
	if left.menus["settings"].blocks["live"].collapsed {
		t.Fatal("forced menu snapshot should reset local collapse state")
	}

	left.updateField("settings", ref("live", "query"), "go")
	left.requestAutocomplete("settings", ref("live", "query"), "go")
	left.updateField("settings", ref("live", "count"), float64(2))
	left.updateField("settings", ref("live", "enabled"), false)
	left.pressButton("settings", "live", "refresh", nil)
	routeFrontend(t, left, backend, 6)
	for range 6 {
		backend.receiveOne(t)
	}
	if len(backend.handled) != 6 {
		t.Fatalf("backend handled %d live messages, want 6", len(backend.handled))
	}

	left.updateField("settings", ref("profile", "email"), "bad@example.com")
	routeFrontend(t, left, backend, 1)
	backend.receiveOne(t)
	left.applyForm("settings", "profile")
	if len(left.outbox) != 0 {
		t.Fatal("frontend applied form before backend ok status")
	}

	backend.send("left", formular.FieldStatusMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageFieldStatus, MenuID: "settings", MenuGeneration: 1, BlockGeneration: 1},
		Field:       ref("profile", "email"),
		Status:      formular.StatusOK,
	})
	deliverBackend(t, backend, left, right, 1)
	if got := left.field("settings", ref("profile", "email")).status; got != "" {
		t.Fatalf("stale status changed frontend field status to %q", got)
	}

	readonly := true
	backend.send("left", formular.FieldStatusMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageFieldStatus, MenuID: "settings", MenuGeneration: 1, BlockGeneration: 3},
		Field:       ref("profile", "email"),
		Status:      formular.StatusOK,
		StatusText:  "accepted",
		Readonly:    &readonly,
	})
	deliverBackend(t, backend, left, right, 1)
	if got := left.field("settings", ref("profile", "email")).status; got != formular.StatusOK {
		t.Fatalf("fresh status was not applied, got %q", got)
	}
	left.applyForm("settings", "profile")
	routeFrontend(t, left, backend, 1)
	backend.receiveOne(t)
	if _, ok := backend.handled[len(backend.handled)-1].(formular.FormApplyMessage); !ok {
		t.Fatalf("last backend message = %T, want form.apply", backend.handled[len(backend.handled)-1])
	}

	right.updateField("settings", ref("upload", "attachment"), "aGVsbG8=")
	right.applyForm("settings", "upload")
	routeFrontend(t, right, backend, 1)
	backend.receiveOne(t)

	backend.send("left", formular.AutocompleteHintsMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageAutocompleteHints, MenuID: "settings", MenuGeneration: 1, BlockGeneration: 2},
		Field:       ref("live", "query"),
		Prefix:      "go",
		Hints:       []string{"rust"},
	})
	backend.send("left", formular.AutocompleteHintsMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageAutocompleteHints, MenuID: "settings", MenuGeneration: 1, BlockGeneration: 2},
		Field:       ref("live", "query"),
		Prefix:      "go",
		Hints:       []string{"go test", "go vet"},
	})
	deliverBackend(t, backend, left, right, 2)
	if !reflect.DeepEqual(left.hints, []string{"go test", "go vet"}) {
		t.Fatalf("autocomplete hints = %#v", left.hints)
	}

	backend.send("right", formular.BlockDeleteMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageBlockDelete, MenuID: "settings", MenuGeneration: 0, BlockGeneration: 0},
		BlockID:     "upload",
	})
	deliverBackend(t, backend, left, right, 1)
	if right.menus["settings"].blocks["upload"] != nil {
		t.Fatal("frontend should honor block deletes even when generations are stale")
	}

	backend.send("right", formular.FieldStatusMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageFieldStatus, MenuID: "missing", MenuGeneration: 1, BlockGeneration: 1},
		Field:       ref("live", "query"),
		Status:      formular.StatusOK,
	})
	backend.send("right", formular.FieldStatusMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageFieldStatus, MenuID: "settings", MenuGeneration: 1, BlockGeneration: 2},
		Field:       formular.FieldRef{BlockID: "live"},
		Status:      "not-a-status",
	})
	backend.send("right", map[string]any{"type": formular.MessageMenuSnapshot})
	deliverBackend(t, backend, left, right, 3)
	if right.ignored != 3 {
		t.Fatalf("right ignored %d malformed/invalid backend messages, want 3", right.ignored)
	}

	left.outbox <- formular.FieldUpdateMessage{
		MessageBase: frontendBase(formular.MessageFieldUpdate, "settings", 0, 2),
		Field:       ref("live", "query"),
		Value:       "stale",
	}
	left.outbox <- formular.FieldUpdateMessage{
		MessageBase: frontendBase(formular.MessageFieldUpdate, "unknown", 1, 2),
		Field:       ref("live", "query"),
		Value:       "unknown",
	}
	left.outbox <- formular.FieldUpdateMessage{
		MessageBase: frontendBase(formular.MessageFieldUpdate, "settings", 1, 2),
		Field:       formular.FieldRef{BlockID: "live"},
		Value:       "invalid",
	}
	left.outbox <- "not a protocol message"
	routeFrontend(t, left, backend, 4)
	for range 4 {
		backend.receiveOne(t)
	}
	if backend.ignored != 4 {
		t.Fatalf("backend ignored %d stale/unknown/malformed frontend messages, want 4", backend.ignored)
	}
}

func TestE2EMenuFixturesExerciseAllProtocolFeatures(t *testing.T) {
	menus := []formular.MenuSnapshotMessage{settingsMenu(), toolsMenu()}
	seenKinds := map[string]bool{}
	seenItems := map[string]bool{}
	seenTextFormats := map[string]bool{}
	var sawForm, sawNonForm, sawInactive, sawCopyable, sawAutocomplete, sawArray, sawNestedButton bool

	for _, menu := range menus {
		if err := menu.Validate(); err != nil {
			t.Fatalf("fixture menu %q failed validation: %v", menu.MenuID, err)
		}
		for _, block := range menu.Blocks {
			sawForm = sawForm || block.Form
			sawNonForm = sawNonForm || !block.Form
			sawInactive = sawInactive || block.Inactive
			sawCopyable = sawCopyable || block.Copyable != nil
			for _, item := range block.Items {
				collectFeatures(item, seenKinds, seenItems, seenTextFormats, &sawAutocomplete, &sawArray, &sawNestedButton)
			}
		}
	}

	for _, kind := range []string{
		formular.FieldText, formular.FieldInt, formular.FieldFloat, formular.FieldFile,
		formular.FieldCheckbox, formular.FieldRadio, formular.FieldRange, formular.FieldArray,
	} {
		if !seenKinds[kind] {
			t.Fatalf("fixtures did not include %s field", kind)
		}
	}
	for _, item := range []string{formular.ItemHeader, formular.ItemLabel, formular.ItemProgressbar, formular.ItemLogs, formular.ItemButton, formular.ItemField} {
		if !seenItems[item] {
			t.Fatalf("fixtures did not include %s item", item)
		}
	}
	for _, format := range []string{formular.TextPlain, formular.TextMarkdown, formular.TextCode} {
		if !seenTextFormats[format] {
			t.Fatalf("fixtures did not include %s label", format)
		}
	}
	if !sawForm || !sawNonForm || !sawInactive || !sawCopyable || !sawAutocomplete || !sawArray || !sawNestedButton {
		t.Fatalf("feature flags: form=%v nonForm=%v inactive=%v copyable=%v autocomplete=%v array=%v nestedButton=%v",
			sawForm, sawNonForm, sawInactive, sawCopyable, sawAutocomplete, sawArray, sawNestedButton)
	}
}

func deliverBackend(t *testing.T, backend *testBackend, left, right *testFrontend, count int) {
	t.Helper()
	for range count {
		select {
		case envelope := <-backend.outbox:
			switch envelope.client {
			case left.name:
				left.inbox <- envelope.msg
				left.receiveOne(t)
			case right.name:
				right.inbox <- envelope.msg
				right.receiveOne(t)
			default:
				t.Fatalf("unknown frontend client %q", envelope.client)
			}
		default:
			t.Fatal("backend outbox was empty")
		}
	}
}

func routeFrontend(t *testing.T, frontend *testFrontend, backend *testBackend, count int) {
	t.Helper()
	for range count {
		select {
		case msg := <-frontend.outbox:
			backend.inbox <- routedMessage{client: frontend.name, msg: msg}
		default:
			t.Fatalf("%s outbox was empty", frontend.name)
		}
	}
}

func newMenuView(snapshot formular.MenuSnapshotMessage) *menuView {
	view := &menuView{generation: snapshot.MenuGeneration, blocks: map[string]*blockView{}}
	for _, block := range snapshot.Blocks {
		view.putBlock(block)
	}
	return view
}

func (m *menuView) putBlock(block formular.Block) {
	view := &blockView{
		block:     block.Copy(),
		fields:    map[fieldKey]*fieldView{},
		collapsed: block.Collapsed,
	}
	for i := range view.block.Items {
		collectFieldViews(block.ID, nil, &view.block.Items[i], view.fields)
	}
	m.blocks[block.ID] = view
	m.rebuildOrder()
}

func (m *menuView) rebuildOrder() {
	m.order = m.order[:0]
	for id := range m.blocks {
		m.order = append(m.order, id)
	}
	sort.Slice(m.order, func(i, j int) bool {
		left, right := m.blocks[m.order[i]].block, m.blocks[m.order[j]].block
		if left.Order == right.Order {
			return left.ID < right.ID
		}
		return left.Order < right.Order
	})
}

func collectFieldViews(blockID string, path []formular.ElementPathSegment, item *formular.Item, fields map[fieldKey]*fieldView) {
	if item.Type != formular.ItemField || item.Field == nil {
		return
	}
	key := fieldKey{blockID: blockID, path: pathKey(path), fieldID: item.ID}
	fields[key] = &fieldView{
		item:       item,
		value:      copyValue(item.Field.Value),
		status:     item.Field.Status,
		statusText: item.Field.StatusText,
		readonly:   item.Field.Readonly,
	}
	if item.Field.Kind != formular.FieldArray {
		return
	}
	for _, elem := range item.Field.Elements {
		elemPath := append(append([]formular.ElementPathSegment(nil), path...), formular.ElementPathSegment{
			ArrayFieldID: item.ID,
			ElementID:    elem.ID,
		})
		for i := range elem.Items {
			collectFieldViews(blockID, elemPath, &elem.Items[i], fields)
		}
	}
}

func collectFeatures(item formular.Item, kinds, items, formats map[string]bool, autocomplete, array, nestedButton *bool) {
	items[item.Type] = true
	if item.Type == formular.ItemLabel {
		format := item.Format
		if format == "" {
			format = formular.TextPlain
		}
		formats[format] = true
	}
	if item.Type != formular.ItemField || item.Field == nil {
		return
	}
	kinds[item.Field.Kind] = true
	*autocomplete = *autocomplete || item.Field.Autocomplete != nil && item.Field.Autocomplete.Enabled
	*array = *array || item.Field.Kind == formular.FieldArray
	for _, tmpl := range item.Field.Templates {
		for _, nested := range tmpl.Items {
			if nested.Type == formular.ItemButton {
				*nestedButton = true
			}
			collectFeatures(nested, kinds, items, formats, autocomplete, array, nestedButton)
		}
	}
	for _, elem := range item.Field.Elements {
		for _, nested := range elem.Items {
			if nested.Type == formular.ItemButton {
				*nestedButton = true
			}
			collectFeatures(nested, kinds, items, formats, autocomplete, array, nestedButton)
		}
	}
}

func validateFrontendMessage(msg any) error {
	switch m := msg.(type) {
	case formular.FieldUpdateMessage:
		return m.Validate()
	case formular.FieldValidateMessage:
		return m.Validate()
	case formular.FormApplyMessage:
		return m.Validate()
	case formular.ButtonPressMessage:
		return m.Validate()
	case formular.AutocompleteRequestMessage:
		return m.Validate()
	default:
		return errMalformed{}
	}
}

type errMalformed struct{}

func (errMalformed) Error() string { return "malformed message" }

func frontendMessageBase(msg any) formular.MessageBase {
	switch m := msg.(type) {
	case formular.FieldUpdateMessage:
		return m.MessageBase
	case formular.FieldValidateMessage:
		return m.MessageBase
	case formular.FormApplyMessage:
		return m.MessageBase
	case formular.ButtonPressMessage:
		return m.MessageBase
	case formular.AutocompleteRequestMessage:
		return m.MessageBase
	default:
		return formular.MessageBase{}
	}
}

func frontendMessageBlockID(msg any) (string, bool) {
	switch m := msg.(type) {
	case formular.FieldUpdateMessage:
		return m.Field.BlockID, true
	case formular.FieldValidateMessage:
		return m.Field.BlockID, true
	case formular.FormApplyMessage:
		return m.BlockID, true
	case formular.ButtonPressMessage:
		return m.BlockID, true
	case formular.AutocompleteRequestMessage:
		return m.Field.BlockID, true
	default:
		return "", false
	}
}

func frontendBase(messageType, menuID string, menuGeneration, blockGeneration uint64) formular.MessageBase {
	return formular.MessageBase{
		Type:            messageType,
		MenuID:          menuID,
		MenuGeneration:  menuGeneration,
		BlockGeneration: blockGeneration,
	}
}

func findBlock(menu formular.MenuSnapshotMessage, blockID string) (formular.Block, bool) {
	for _, block := range menu.Blocks {
		if block.ID == blockID {
			return block, true
		}
	}
	return formular.Block{}, false
}

func sameFieldRef(left, right formular.FieldRef) bool {
	return left.BlockID == right.BlockID && left.FieldID == right.FieldID && reflect.DeepEqual(left.ElementPath, right.ElementPath)
}

func keyFromRef(ref formular.FieldRef) fieldKey {
	return fieldKey{blockID: ref.BlockID, path: pathKey(ref.ElementPath), fieldID: ref.FieldID}
}

func pathKey(path []formular.ElementPathSegment) string {
	if len(path) == 0 {
		return ""
	}
	parts := make([]string, len(path))
	for i, segment := range path {
		parts[i] = segment.ArrayFieldID + "/" + segment.ElementID
	}
	return strings.Join(parts, "|")
}

func ref(blockID, fieldID string) formular.FieldRef {
	return formular.FieldRef{BlockID: blockID, FieldID: fieldID}
}

func copyValue(value any) any {
	switch v := value.(type) {
	case []formular.ArrayElementValue:
		out := make([]formular.ArrayElementValue, len(v))
		for i := range v {
			out[i] = v[i].Copy()
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, value := range v {
			out[k] = copyValue(value)
		}
		return out
	default:
		return v
	}
}

func emptyValue(value any) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return v == ""
	case []formular.ArrayElementValue:
		return len(v) == 0
	default:
		return false
	}
}

func buttonActive(block formular.Block, buttonID string, path []formular.ElementPathSegment) bool {
	for _, item := range block.Items {
		if item.Type == formular.ItemButton && item.ID == buttonID {
			return !item.Inactive
		}
		if item.Type != formular.ItemField || item.Field == nil || item.Field.Kind != formular.FieldArray {
			continue
		}
		for _, elem := range item.Field.Elements {
			if len(path) == 1 && path[0].ArrayFieldID == item.ID && path[0].ElementID == elem.ID {
				for _, nested := range elem.Items {
					if nested.Type == formular.ItemButton && nested.ID == buttonID {
						return !nested.Inactive
					}
				}
			}
		}
	}
	return false
}

func assertBlocks(t *testing.T, frontend *testFrontend, menuID string, want []string) {
	t.Helper()
	if got := frontend.menus[menuID].order; !reflect.DeepEqual(got, want) {
		t.Fatalf("%s blocks = %#v, want %#v", frontend.name, got, want)
	}
}

func settingsMenu() formular.MenuSnapshotMessage {
	minZero := 0.0
	maxTen := 10.0
	fraction := uint(2)
	maxBytes := uint64(2048)
	progress := uint(35)
	return formular.MenuSnapshotMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageMenuSnapshot, MenuID: "settings", MenuGeneration: 1},
		Blocks: []formular.Block{
			{
				ID:          "live",
				Order:       10,
				Generation:  2,
				Collapsible: true,
				Copyable:    &formular.Copyable{Text: "live block"},
				Items: []formular.Item{
					{Type: formular.ItemHeader, ID: "live-title", Text: "Live Controls", Help: "Realtime controls"},
					{Type: formular.ItemLabel, ID: "live-help", Text: "Use **carefully**", Format: formular.TextMarkdown},
					{Type: formular.ItemProgressbar, ID: "live-progress", Label: "Live progress", Progress: &progress},
					{Type: formular.ItemLogs, ID: "live-logs", Label: "Live logs", Logs: []formular.LogLine{
						{Level: formular.LogTrace, Text: "trace line"},
						{Level: formular.LogDebug, Text: "debug line"},
						{Level: formular.LogInfo, Text: "info line"},
						{Level: formular.LogWarn, Text: "warn line"},
						{Level: formular.LogError, Text: "error line"},
						{Level: formular.LogPanic, Text: "panic line"},
					}},
					{Type: formular.ItemField, ID: "query", Label: "Query", Field: &formular.Field{
						Kind:          formular.FieldText,
						Value:         "",
						Placeholder:   "search",
						Validation:    true,
						Autocomplete:  &formular.Autocomplete{Enabled: true, Tag: "queries"},
						AllowedValues: []any{"go", "rust"},
					}},
					{Type: formular.ItemField, ID: "count", Label: "Count", Field: &formular.Field{
						Kind:          formular.FieldInt,
						Value:         float64(1),
						Min:           &minZero,
						Max:           &maxTen,
						AllowedValues: []any{float64(1), float64(2)},
					}},
					{Type: formular.ItemField, ID: "enabled", Label: "Enabled", Field: &formular.Field{Kind: formular.FieldCheckbox, Value: true}},
					{Type: formular.ItemButton, ID: "refresh", Label: "Refresh"},
					{Type: formular.ItemButton, ID: "disabled-action", Label: "Disabled", Inactive: true},
				},
			},
			{
				ID:         "profile",
				Order:      20,
				Generation: 3,
				Form:       true,
				Items: []formular.Item{
					{Type: formular.ItemLabel, ID: "snippet", Text: "fmt.Println(\"ok\")", Format: formular.TextCode, Syntax: "go"},
					{Type: formular.ItemField, ID: "email", Label: "Email", Field: &formular.Field{
						Kind:       formular.FieldText,
						Value:      "",
						Required:   true,
						Validation: true,
						Subtype:    "email",
					}},
					{Type: formular.ItemField, ID: "password", Label: "Password", Field: &formular.Field{
						Kind:     formular.FieldText,
						Value:    "secret",
						Secret:   true,
						Readonly: true,
					}},
					{Type: formular.ItemField, ID: "bio", Label: "Bio", Field: &formular.Field{
						Kind:      formular.FieldText,
						Value:     "hello\nworld",
						Multiline: true,
					}},
					{Type: formular.ItemField, ID: "ratio", Label: "Ratio", Field: &formular.Field{
						Kind:     formular.FieldFloat,
						Value:    1.25,
						Min:      &minZero,
						Max:      &maxTen,
						Fraction: &fraction,
					}},
					{Type: formular.ItemField, ID: "mode", Label: "Mode", Field: &formular.Field{
						Kind:          formular.FieldRadio,
						Value:         "auto",
						AllowedValues: []any{"auto", "manual"},
					}},
					{Type: formular.ItemField, ID: "level", Label: "Level", Field: &formular.Field{
						Kind:  formular.FieldRange,
						Value: 5.0,
						Min:   &minZero,
						Max:   &maxTen,
					}},
					{Type: formular.ItemField, ID: "credentials", Label: "Credentials", Field: credentialsField()},
				},
			},
			{
				ID:         "upload",
				Order:      30,
				Generation: 1,
				Form:       true,
				Items: []formular.Item{
					{Type: formular.ItemField, ID: "attachment", Label: "Attachment", Field: &formular.Field{
						Kind:     formular.FieldFile,
						Value:    "aGVsbG8=",
						MaxBytes: &maxBytes,
						Accept:   []string{"text/plain"},
					}},
				},
			},
			{
				ID:         "disabled",
				Order:      40,
				Generation: 1,
				Inactive:   true,
				Items: []formular.Item{
					{Type: formular.ItemField, ID: "ignored", Label: "Ignored", Field: &formular.Field{Kind: formular.FieldText, Value: "no input"}},
					{Type: formular.ItemButton, ID: "no-op", Label: "No-op"},
				},
			},
		},
	}
}

func toolsMenu() formular.MenuSnapshotMessage {
	return formular.MenuSnapshotMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageMenuSnapshot, MenuID: "tools", MenuGeneration: 7},
		Blocks: []formular.Block{
			{
				ID:         "plain",
				Order:      1,
				Generation: 1,
				Items: []formular.Item{
					{Type: formular.ItemLabel, ID: "plain-label", Text: "plain text", Format: formular.TextPlain},
				},
			},
		},
	}
}

func credentialsField() *formular.Field {
	return &formular.Field{
		Kind: formular.FieldArray,
		Value: []formular.ArrayElementValue{
			{ID: "token-1", Template: "token", Values: map[string]any{"name": "primary", "token": "abc"}},
		},
		Templates: []formular.ArrayTemplate{
			{
				Name:  "token",
				Label: "Token",
				Items: []formular.Item{
					{Type: formular.ItemLabel, ID: "token-help", Text: "Token row"},
					{Type: formular.ItemField, ID: "name", Label: "Name", Field: &formular.Field{Kind: formular.FieldText, Required: true}},
					{Type: formular.ItemField, ID: "token", Label: "Token", Field: &formular.Field{Kind: formular.FieldText, Secret: true}},
					{Type: formular.ItemButton, ID: "rotate", Label: "Rotate"},
				},
			},
			{
				Name:  "note",
				Label: "Note",
				Items: []formular.Item{
					{Type: formular.ItemLabel, ID: "note-help", Text: "Note row"},
					{Type: formular.ItemField, ID: "text", Label: "Text", Field: &formular.Field{Kind: formular.FieldText, Multiline: true}},
				},
			},
		},
		Elements: []formular.ArrayElement{
			{
				ID:       "token-1",
				Template: "token",
				Copyable: &formular.Copyable{Text: "abc"},
				Items: []formular.Item{
					{Type: formular.ItemLabel, ID: "token-help", Text: "Token row"},
					{Type: formular.ItemField, ID: "name", Label: "Name", Field: &formular.Field{Kind: formular.FieldText, Value: "primary", Required: true}},
					{Type: formular.ItemField, ID: "token", Label: "Token", Field: &formular.Field{Kind: formular.FieldText, Value: "abc", Secret: true}},
					{Type: formular.ItemButton, ID: "rotate", Label: "Rotate"},
				},
			},
		},
	}
}
