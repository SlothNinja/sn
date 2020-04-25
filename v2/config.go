package sn

const (
	contestKind       = "Contest"
	mlogKind          = "MessageLog"
	mlogKey           = "MessageLog"
	homePath          = "/"
	msgEnter          = "Entering"
	msgExit           = "Exiting"
	currentRatingsKey = "CurrentRatings"
	projectedKey      = "Projected"
	typeKey           = "Type"
)

type ActionType int

const (
	ActionNone ActionType = iota
	ActionSave
	ActionSaveAndStatUpdate
	ActionCache
	ActionUndoAdd
	ActionUndoReplace
	ActionUndoPop
	ActionUndo
	ActionRedo
	ActionReset
)
