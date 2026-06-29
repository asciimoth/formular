package formular

// MenuSnapshot constructs a complete menu.snapshot message.
//
// The blocks are deep-copied so callers can continue mutating local menu
// definitions after constructing the outbound message.
func MenuSnapshot(menuID string, generation uint64, blocks ...Block) MenuSnapshotMessage {
	return MenuSnapshotMessage{
		MessageBase: MessageBase{
			Type:           MessageMenuSnapshot,
			MenuID:         menuID,
			MenuGeneration: generation,
		},
		Blocks: copyBlocks(blocks),
	}
}

// ForcedMenuSnapshot constructs a menu.snapshot message that requests a full
// frontend refresh, including local state that ordinary snapshots may preserve.
func ForcedMenuSnapshot(menuID string, generation uint64, blocks ...Block) MenuSnapshotMessage {
	msg := MenuSnapshot(menuID, generation, blocks...)
	msg.Force = true
	return msg
}

// BlockSnapshot constructs a block.snapshot message for one block.
//
// menuGen is the containing menu generation. blockGen is copied into the
// message envelope; the block's own Generation field is left unchanged so
// callers retain explicit control of the block snapshot identity.
func BlockSnapshot(menuID string, menuGen, blockGen uint64, block Block) BlockSnapshotMessage {
	return BlockSnapshotMessage{
		MessageBase: MessageBase{
			Type:            MessageBlockSnapshot,
			MenuID:          menuID,
			MenuGeneration:  menuGen,
			BlockGeneration: blockGen,
		},
		Block: block.Copy(),
	}
}

// FieldStatusFromError constructs a field.status message from a Go error.
//
// A nil error produces an ok status. A non-nil error produces an error status
// with err.Error() as StatusText.
func FieldStatusFromError(menuID string, menuGen, blockGen uint64, ref FieldRef, err error) FieldStatusMessage {
	msg := FieldStatusMessage{
		MessageBase: MessageBase{
			Type:            MessageFieldStatus,
			MenuID:          menuID,
			MenuGeneration:  menuGen,
			BlockGeneration: blockGen,
		},
		Field:  ref.Copy(),
		Status: StatusOK,
	}
	if err != nil {
		msg.Status = StatusError
		msg.StatusText = err.Error()
	}
	return msg
}

// AutocompleteHints constructs an autocomplete.hints message for one field.
func AutocompleteHints(menuID string, menuGen uint64, ref FieldRef, prefix string, hints []string) AutocompleteHintsMessage {
	return AutocompleteHintsMessage{
		MessageBase: MessageBase{
			Type:           MessageAutocompleteHints,
			MenuID:         menuID,
			MenuGeneration: menuGen,
		},
		Field:  ref.Copy(),
		Prefix: prefix,
		Hints:  copySlice(hints),
	}
}
