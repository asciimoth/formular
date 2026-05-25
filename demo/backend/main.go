//go:build js && wasm

package main

import (
	"encoding/json"
	"strings"
	"syscall/js"
	"time"

	formular "github.com/asciimoth/formular"
)

type serverState struct {
	ID       string
	Template string
	Values   map[string]any
}

var profileValues = map[string]any{
	"name":     "Ada",
	"email":    "admin@example.com",
	"timezone": "UTC",
	"password": "",
	"bio":      "Line one\nLine two",
	"age":      37,
	"score":    98.5,
	"avatar":   nil,
}

var liveValues = map[string]any{
	"enabled":          true,
	"mode":             "balanced",
	"volume":           42,
	"manualInput":      "Change the radio below",
	"manualStatusMode": "unset",
}

var progressValue uint
var instanceID int
var generatedServerCounter int
var logLines = []formular.LogLine{
	{Level: formular.LogInfo, Text: "Demo backend initialized"},
	{Level: formular.LogDebug, Text: "Waiting for form submissions"},
}

var serverValues = []serverState{{
	ID:       "server-1",
	Template: "http",
	Values: map[string]any{
		"host": "localhost",
		"port": 8080,
	},
}}

func main() {
	done := make(chan struct{})
	instanceID = nextInstanceID()
	js.Global().Set("formularBackendSend", js.FuncOf(receive))
	sendSnapshots()
	go progressLoop(instanceID)
	<-done
}

func nextInstanceID() int {
	current := js.Global().Get("formularBackendInstance")
	next := 1
	if current.Type() == js.TypeNumber {
		next = current.Int() + 1
	}
	js.Global().Set("formularBackendInstance", next)
	return next
}

func currentInstance(id int) bool {
	current := js.Global().Get("formularBackendInstance")
	return current.Type() == js.TypeNumber && current.Int() == id
}

func receive(this js.Value, args []js.Value) any {
	if len(args) == 0 {
		return nil
	}
	var msg map[string]any
	if err := json.Unmarshal([]byte(args[0].String()), &msg); err != nil {
		return nil
	}
	menuID, _ := msg["menuId"].(string)
	switch msg["type"] {
	case "demo.snapshot.request":
		sendSnapshots()
	case formular.MessageFieldValidate:
		validate(menuID, msg)
	case formular.MessageAutocompleteRequest:
		autocomplete(menuID, msg)
	case formular.MessageButtonPress:
		button(menuID, msg)
	case formular.MessageFormApply:
		applyForm(menuID, msg)
	case formular.MessageFieldUpdate:
		updateField(menuID, msg)
		ack(menuID, "live", "Realtime update received")
	}
	return nil
}

func sendSnapshots() {
	send(snapshot("left", 1, leftBlocks()))
	send(snapshot("right", 1, rightBlocks()))
}

func send(v any) {
	if !currentInstance(instanceID) {
		return
	}
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	js.Global().Call("formularFrontendDispatch", string(data))
}

func progressLoop(id int) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if !currentInstance(id) {
			return
		}
		if progressValue >= 100 {
			progressValue = 0
		} else {
			progressValue += 10
		}
		send(formular.BlockSnapshotMessage{
			MessageBase: formular.MessageBase{Type: formular.MessageBlockSnapshot, MenuID: "left", MenuGeneration: 1, BlockGeneration: 1},
			Block:       leftBlocks()[0],
		})
	}
}

func snapshot(menuID string, generation uint64, blocks []formular.Block) formular.MenuSnapshotMessage {
	return formular.MenuSnapshotMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageMenuSnapshot, MenuID: menuID, MenuGeneration: generation},
		Force:       true,
		Blocks:      blocks,
	}
}

func validate(menuID string, msg map[string]any) {
	field := fieldRef(msg)
	value := ""
	if raw, ok := msg["value"].(string); ok {
		value = raw
	}
	status := formular.StatusOK
	statusText := "Looks good"
	if field.FieldID == "email" && !strings.Contains(value, "@") {
		status = formular.StatusError
		statusText = "Email must contain @"
	}
	if field.FieldID == "name" && strings.TrimSpace(value) == "" {
		status = formular.StatusError
		statusText = "Name is required"
	}
	send(formular.FieldStatusMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageFieldStatus, MenuID: menuID, MenuGeneration: 1, BlockGeneration: 1},
		Field:       field,
		Status:      status,
		StatusText:  statusText,
	})
}

func autocomplete(menuID string, msg map[string]any) {
	field := fieldRef(msg)
	prefix, _ := msg["prefix"].(string)
	all := autocompleteValues(field.FieldID)
	hints := make([]string, 0, len(all))
	for _, hint := range all {
		if strings.HasPrefix(hint, prefix) {
			hints = append(hints, hint)
		}
	}
	send(formular.AutocompleteHintsMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageAutocompleteHints, MenuID: menuID, MenuGeneration: 1, BlockGeneration: 1},
		Field:       field,
		Prefix:      prefix,
		Hints:       hints,
	})
}

func autocompleteValues(fieldID string) []string {
	switch fieldID {
	case "email":
		return []string{"admin@example.com", "author@example.com", "billing@example.com", "support@example.com"}
	case "timezone":
		return []string{"UTC", "Europe/Tbilisi", "Europe/Berlin", "America/New_York", "America/Los_Angeles", "Asia/Tokyo"}
	default:
		return nil
	}
}

func button(menuID string, msg map[string]any) {
	buttonID, _ := msg["buttonId"].(string)
	if buttonID == "refresh" {
		ack(menuID, "live", "Refresh button pressed")
		return
	}
	if menuID == "right" && buttonID == "generate" {
		generateServerElement(msg)
		return
	}
	ack(menuID, "profile", "Button "+buttonID+" pressed")
}

func generateServerElement(msg map[string]any) {
	path := elementPath(msg)
	if len(path) == 0 {
		return
	}
	segment := path[len(path)-1]
	if segment.ArrayFieldID != "servers" {
		return
	}
	for i := range serverValues {
		if serverValues[i].ID != segment.ElementID {
			continue
		}
		generatedServerCounter++
		serverValues[i].Values = generatedServerValues(serverValues[i].Template, generatedServerCounter)
		send(formular.BlockSnapshotMessage{
			MessageBase: formular.MessageBase{Type: formular.MessageBlockSnapshot, MenuID: "right", MenuGeneration: 1, BlockGeneration: 1},
			Block:       rightBlocks()[0],
		})
		return
	}
}

func generatedServerValues(template string, counter int) map[string]any {
	switch template {
	case "database":
		drivers := []any{"postgres", "mysql", "sqlite"}
		return map[string]any{
			"driver": drivers[counter%len(drivers)],
			"dsn":    "postgres://generated-" + jsonNumberText(counter) + ".local/app",
			"pool":   5 + counter,
		}
	case "queue":
		brokers := []any{"nats", "redis", "rabbitmq"}
		return map[string]any{
			"broker":  brokers[counter%len(brokers)],
			"subject": "events.generated." + jsonNumberText(counter),
			"durable": counter%2 == 0,
		}
	default:
		return map[string]any{
			"host": "generated-" + jsonNumberText(counter) + ".local",
			"port": 8000 + counter,
			"tls":  counter%2 == 0,
		}
	}
}

func ack(menuID, blockID, text string) {
	send(formular.BlockSnapshotMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageBlockSnapshot, MenuID: menuID, MenuGeneration: 1, BlockGeneration: 2},
		Block: formular.Block{
			ID:         blockID + "-status",
			Order:      99,
			Generation: 2,
			Form:       false,
			Items: []formular.Item{
				{Type: formular.ItemLabel, ID: "status", Text: text, Format: formular.TextPlain},
			},
		},
	})
}

func fieldRef(msg map[string]any) formular.FieldRef {
	raw, _ := msg["field"].(map[string]any)
	ref := formular.FieldRef{}
	ref.BlockID, _ = raw["blockId"].(string)
	ref.FieldID, _ = raw["fieldId"].(string)
	ref.ElementPath = parseElementPath(raw["elementPath"])
	return ref
}

func elementPath(msg map[string]any) []formular.ElementPathSegment {
	return parseElementPath(msg["elementPath"])
}

func parseElementPath(raw any) []formular.ElementPathSegment {
	var out []formular.ElementPathSegment
	path, ok := raw.([]any)
	if !ok {
		return out
	}
	for _, item := range path {
		segRaw, _ := item.(map[string]any)
		seg := formular.ElementPathSegment{}
		seg.ArrayFieldID, _ = segRaw["arrayFieldId"].(string)
		seg.ElementID, _ = segRaw["elementId"].(string)
		out = append(out, seg)
	}
	return out
}

func applyForm(menuID string, msg map[string]any) {
	if menuID != "left" {
		return
	}
	blockID, _ := msg["blockId"].(string)
	values, ok := msg["values"].(map[string]any)
	if !ok {
		return
	}
	if blockID == "log-submit" {
		appendSubmittedLog(values)
		ack(menuID, "log-submit", "Log line submitted")
		return
	}
	for key, value := range values {
		profileValues[key] = value
	}
	ack(menuID, "profile", "Form values accepted by Go WASM backend")
}

func appendSubmittedLog(values map[string]any) {
	level, _ := values["level"].(string)
	if !validLogLevel(level) {
		level = formular.LogInfo
	}
	message, _ := values["message"].(string)
	message = strings.TrimSpace(message)
	if message == "" {
		message = "Submitted log line"
	}
	logLines = append(logLines, formular.LogLine{Level: level, Text: message})
	send(formular.BlockSnapshotMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageBlockSnapshot, MenuID: "right", MenuGeneration: 1, BlockGeneration: 1},
		Block:       rightBlocks()[0],
	})
}

func validLogLevel(level string) bool {
	switch level {
	case formular.LogTrace, formular.LogDebug, formular.LogInfo, formular.LogWarn, formular.LogError, formular.LogPanic:
		return true
	default:
		return false
	}
}

func updateField(menuID string, msg map[string]any) {
	if menuID != "right" {
		return
	}
	ref := fieldRef(msg)
	value, ok := msg["value"]
	if !ok {
		return
	}
	if len(ref.ElementPath) == 0 {
		if ref.FieldID == "servers" {
			updateServers(value)
			return
		}
		liveValues[ref.FieldID] = value
		if ref.FieldID == "manualStatusMode" {
			sendManualStatus(menuID)
		}
		return
	}
	segment := ref.ElementPath[len(ref.ElementPath)-1]
	if segment.ArrayFieldID != "servers" {
		return
	}
	for i := range serverValues {
		if serverValues[i].ID == segment.ElementID {
			serverValues[i].Values[ref.FieldID] = value
			return
		}
	}
}

func sendManualStatus(menuID string) {
	status, _ := liveValues["manualStatusMode"].(string)
	if status == "" {
		status = formular.StatusUnset
	}
	send(formular.FieldStatusMessage{
		MessageBase: formular.MessageBase{Type: formular.MessageFieldStatus, MenuID: menuID, MenuGeneration: 1, BlockGeneration: 1},
		Field:       formular.FieldRef{BlockID: "live", FieldID: "manualInput"},
		Status:      status,
		StatusText:  manualStatusText(status),
	})
}

func manualStatusText(status string) string {
	switch status {
	case formular.StatusOK:
		return "Backend marked this field as OK"
	case formular.StatusWarn:
		return "Backend marked this field as a warning"
	case formular.StatusError:
		return "Backend marked this field as an error"
	default:
		return "Backend status is unset"
	}
}

func updateServers(value any) {
	raw, ok := value.([]any)
	if !ok {
		return
	}
	next := make([]serverState, 0, len(raw))
	for _, item := range raw {
		element, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id, _ := element["id"].(string)
		template, _ := element["template"].(string)
		values, _ := element["values"].(map[string]any)
		if id == "" || template == "" {
			continue
		}
		if values == nil {
			values = map[string]any{}
		}
		next = append(next, serverState{ID: id, Template: template, Values: values})
	}
	serverValues = next
}

func profileValue(id string, fallback any) any {
	if value, ok := profileValues[id]; ok {
		return value
	}
	return fallback
}

func liveValue(id string, fallback any) any {
	if value, ok := liveValues[id]; ok {
		return value
	}
	return fallback
}

func leftBlocks() []formular.Block {
	return []formular.Block{
		{
			ID:          "profile",
			Order:       10,
			Generation:  1,
			Form:        true,
			Collapsible: true,
			Copyable:    &formular.Copyable{Text: "profile block"},
			Items: []formular.Item{
				{Type: formular.ItemHeader, ID: "title", Text: "Profile form", Help: "A form block submits all profile fields together with Apply."},
				{Type: formular.ItemLabel, ID: "intro", Text: "Markdown **label** with `code` and [link](https://example.com).", Format: formular.TextMarkdown, Help: "Labels can render plain text, markdown, or code while keeping backend text sanitized."},
				{Type: formular.ItemProgressbar, ID: "sync-progress", Label: "Background sync", Progress: &progressValue, Help: "The Go backend updates this progress bar once per second."},
				withHelp(field("name", formular.FieldText, "Name", profileValue("name", "Ada"), func(f *formular.Field) {
					f.Required = true
					f.Validation = true
					f.Placeholder = "Display name"
				}), "Required text field with backend validation."),
				withHelp(field("email", formular.FieldText, "Email", profileValue("email", "admin@example.com"), func(f *formular.Field) {
					f.Subtype = "email"
					f.Required = true
					f.Validation = true
					f.Autocomplete = &formular.Autocomplete{Enabled: true, Tag: "email"}
				}), "Email input with validation and autocomplete hints from the backend."),
				withHelp(field("timezone", formular.FieldText, "Timezone", profileValue("timezone", "UTC"), func(f *formular.Field) {
					f.Placeholder = "IANA timezone"
					f.Autocomplete = &formular.Autocomplete{Enabled: true, Tag: "timezone"}
				}), "Text field that requests timezone completions while focused or edited."),
				withHelp(field("password", formular.FieldText, "Secret", profileValue("password", ""), func(f *formular.Field) { f.Secret = true }), "Secret text is rendered with a password input."),
				withHelp(field("bio", formular.FieldText, "Bio", profileValue("bio", "Line one\nLine two"), func(f *formular.Field) { f.Multiline = true }), "Multiline text is preserved when the form is applied."),
				withHelp(field("age", formular.FieldInt, "Age", profileValue("age", 37), func(f *formular.Field) {
					min, max := 0.0, 130.0
					f.Min, f.Max = &min, &max
				}), "Integer input constrained to a realistic age range."),
				withHelp(field("score", formular.FieldFloat, "Score", profileValue("score", 98.5), func(f *formular.Field) {
					fraction := uint(2)
					f.Fraction = &fraction
				}), "Float input configured for two fractional digits."),
				withHelp(field("avatar", formular.FieldFile, "Avatar file", profileValue("avatar", nil), func(f *formular.Field) {
					maxBytes := uint64(4098)
					f.MaxBytes = &maxBytes
					f.Accept = []string{"image/png", "image/jpeg"}
				}), "File input limited to small PNG or JPEG payloads for the demo."),
			},
		},
		{
			ID:         "log-submit",
			Order:      20,
			Generation: 1,
			Form:       true,
			Items: []formular.Item{
				{Type: formular.ItemHeader, ID: "title", Text: "Submit log", Help: "This form appends a new line to the Activity log on the right."},
				withHelp(field("level", formular.FieldRadio, "Level", formular.LogInfo, func(f *formular.Field) {
					f.AllowedValues = []any{
						formular.LogTrace,
						formular.LogDebug,
						formular.LogInfo,
						formular.LogWarn,
						formular.LogError,
						formular.LogPanic,
					}
				}), "Radio fields restrict the value to one of the declared log levels."),
				withHelp(field("message", formular.FieldText, "Message", "User submitted log line", func(f *formular.Field) {
					f.Required = true
				}), "Required message text used as the body of the submitted log line."),
			},
		},
	}
}

func rightBlocks() []formular.Block {
	min, max := 0.0, 100.0
	templates := serverTemplates()
	return []formular.Block{
		{
			ID:         "live",
			Order:      10,
			Generation: 1,
			Form:       false,
			Items: []formular.Item{
				{Type: formular.ItemHeader, ID: "title", Text: "Realtime controls", Help: "Controls in this block send field.update messages as they change."},
				withHelp(field("enabled", formular.FieldCheckbox, "Enabled", liveValue("enabled", true), nil), "Checkbox values are sent as booleans immediately."),
				withHelp(field("mode", formular.FieldRadio, "Mode", liveValue("mode", "balanced"), func(f *formular.Field) {
					f.AllowedValues = []any{"fast", "balanced", "safe"}
				}), "Radio options demonstrate allowed values for realtime fields."),
				withHelp(field("volume", formular.FieldRange, "Volume", liveValue("volume", 42), func(f *formular.Field) {
					f.Min, f.Max = &min, &max
				}), "Range fields use min and max constraints from the backend snapshot."),
				withHelp(field("manualInput", formular.FieldText, "Backend validated input", liveValue("manualInput", "Change the radio below"), func(f *formular.Field) {
					status, _ := liveValue("manualStatusMode", formular.StatusUnset).(string)
					f.Status = status
					f.StatusText = manualStatusText(status)
				}), "The status text changes when the Backend status radio is changed."),
				withHelp(field("manualStatusMode", formular.FieldRadio, "Backend status", liveValue("manualStatusMode", formular.StatusUnset), func(f *formular.Field) {
					f.AllowedValues = []any{formular.StatusUnset, formular.StatusOK, formular.StatusWarn, formular.StatusError}
				}), "Changing this field asks the backend to send a field.status update."),
				{Type: formular.ItemButton, ID: "refresh", Label: "Refresh", Help: "Buttons emit button.press messages with the current menu and block generation."},
				{Type: formular.ItemLabel, ID: "code", Text: "go test ./...", Format: formular.TextCode, Syntax: "sh", Help: "Code labels preserve whitespace and display monospace content."},
				{Type: formular.ItemLogs, ID: "activity-log", Label: "Activity log", Logs: logLines, Help: "Log items render structured log lines with level-specific styling."},
				withHelp(field("servers", formular.FieldArray, "Servers", nil, func(f *formular.Field) {
					f.Templates = templates
					f.Elements = serverElements()
					f.Copyable = &formular.Copyable{Text: serversCopyText()}
				}), "Array fields manage repeatable elements created from backend-declared templates."),
			},
		},
	}
}

func serversCopyText() string {
	values := make([]formular.ArrayElementValue, 0, len(serverValues))
	for _, server := range serverValues {
		values = append(values, formular.ArrayElementValue{
			ID:       server.ID,
			Template: server.Template,
			Values:   server.Values,
		})
	}
	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return "[]"
	}
	return string(data)
}

func serverTemplates() []formular.ArrayTemplate {
	return []formular.ArrayTemplate{
		{
			Name:  "http",
			Label: "HTTP server",
			Items: []formular.Item{
				withHelp(field("host", formular.FieldText, "Host", "localhost", nil), "Hostname used by new HTTP server elements."),
				withHelp(field("port", formular.FieldInt, "Port", 8080, nil), "Integer port for the HTTP endpoint."),
				withHelp(field("tls", formular.FieldCheckbox, "TLS", false, nil), "Toggles whether the endpoint should use TLS."),
				{Type: formular.ItemButton, ID: "generate", Label: "Generate", Help: "Asks the backend to generate fresh values for this element."},
				{Type: formular.ItemButton, ID: "ping", Label: "Ping", Help: "Nested buttons include their array element path in button.press messages."},
			},
		},
		{
			Name:  "database",
			Label: "Database",
			Items: []formular.Item{
				withHelp(field("driver", formular.FieldRadio, "Driver", "postgres", func(f *formular.Field) {
					f.AllowedValues = []any{"postgres", "mysql", "sqlite"}
				}), "Database driver choice for new database elements."),
				withHelp(field("dsn", formular.FieldText, "DSN", "postgres://localhost/app", nil), "Connection string for the database element."),
				withHelp(field("pool", formular.FieldInt, "Pool size", 10, nil), "Maximum connection pool size."),
				{Type: formular.ItemButton, ID: "generate", Label: "Generate", Help: "Asks the backend to generate fresh values for this element."},
				{Type: formular.ItemButton, ID: "test", Label: "Test connection", Help: "Nested action for the selected database element."},
			},
		},
		{
			Name:  "queue",
			Label: "Queue",
			Items: []formular.Item{
				withHelp(field("broker", formular.FieldRadio, "Broker", "nats", func(f *formular.Field) {
					f.AllowedValues = []any{"nats", "redis", "rabbitmq"}
				}), "Broker choice for new queue elements."),
				withHelp(field("subject", formular.FieldText, "Subject", "events.created", nil), "Subject, topic, or routing key consumed by the queue."),
				withHelp(field("durable", formular.FieldCheckbox, "Durable", true, nil), "Whether queue state should survive broker restarts."),
				{Type: formular.ItemButton, ID: "generate", Label: "Generate", Help: "Asks the backend to generate fresh values for this element."},
				{Type: formular.ItemButton, ID: "drain", Label: "Drain", Help: "Nested action for the selected queue element."},
			},
		},
	}
}

func serverElements() []formular.ArrayElement {
	out := make([]formular.ArrayElement, 0, len(serverValues))
	for _, server := range serverValues {
		host, _ := server.Values["host"].(string)
		copyText := host
		if port, ok := server.Values["port"]; ok {
			copyText = copyText + ":" + strings.TrimSuffix(strings.TrimSuffix(jsonNumberText(port), ".0"), ".")
		}
		out = append(out, formular.ArrayElement{
			ID:       server.ID,
			Template: server.Template,
			Copyable: &formular.Copyable{Text: copyText},
			Items:    serverElementItems(server),
		})
	}
	return out
}

func serverElementItems(server serverState) []formular.Item {
	switch server.Template {
	case "database":
		return []formular.Item{
			withHelp(field("driver", formular.FieldRadio, "Driver", serverValue(server, "driver", "postgres"), func(f *formular.Field) {
				f.AllowedValues = []any{"postgres", "mysql", "sqlite"}
			}), "Database driver for this element."),
			withHelp(field("dsn", formular.FieldText, "DSN", serverValue(server, "dsn", "postgres://localhost/app"), nil), "Connection string for this database element."),
			withHelp(field("pool", formular.FieldInt, "Pool size", serverValue(server, "pool", 10), nil), "Maximum pool size for this database element."),
			{Type: formular.ItemButton, ID: "generate", Label: "Generate", Help: "Generates fresh database values for this element."},
			{Type: formular.ItemButton, ID: "test", Label: "Test connection", Help: "Sends a nested button press for this database element."},
		}
	case "queue":
		return []formular.Item{
			withHelp(field("broker", formular.FieldRadio, "Broker", serverValue(server, "broker", "nats"), func(f *formular.Field) {
				f.AllowedValues = []any{"nats", "redis", "rabbitmq"}
			}), "Broker for this queue element."),
			withHelp(field("subject", formular.FieldText, "Subject", serverValue(server, "subject", "events.created"), nil), "Subject, topic, or routing key for this element."),
			withHelp(field("durable", formular.FieldCheckbox, "Durable", serverValue(server, "durable", true), nil), "Whether this queue element should be durable."),
			{Type: formular.ItemButton, ID: "generate", Label: "Generate", Help: "Generates fresh queue values for this element."},
			{Type: formular.ItemButton, ID: "drain", Label: "Drain", Help: "Sends a nested button press for this queue element."},
		}
	default:
		return []formular.Item{
			withHelp(field("host", formular.FieldText, "Host", serverValue(server, "host", "localhost"), nil), "Host for this HTTP server element."),
			withHelp(field("port", formular.FieldInt, "Port", serverValue(server, "port", 8080), nil), "Port for this HTTP server element."),
			withHelp(field("tls", formular.FieldCheckbox, "TLS", serverValue(server, "tls", false), nil), "Whether this HTTP server element uses TLS."),
			{Type: formular.ItemButton, ID: "generate", Label: "Generate", Help: "Generates fresh HTTP server values for this element."},
			{Type: formular.ItemButton, ID: "ping", Label: "Ping", Help: "Sends a nested button press for this HTTP server element."},
		}
	}
}

func serverValue(server serverState, id string, fallback any) any {
	if value, ok := server.Values[id]; ok {
		return value
	}
	return fallback
}

func jsonNumberText(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func field(id, kind, label string, value any, configure func(*formular.Field)) formular.Item {
	item := formular.Item{
		Type:  formular.ItemField,
		ID:    id,
		Label: label,
		Field: &formular.Field{Kind: kind, Value: value},
	}
	if configure != nil {
		configure(item.Field)
	}
	return item
}

func withHelp(item formular.Item, help string) formular.Item {
	item.Help = help
	return item
}
