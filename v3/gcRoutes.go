package sn

// AddRoutes adds routing for game.
func (cl *GameClient[GT, G]) addRoutes(prefix string) *GameClient[GT, G] {
	/////////////////////////////////////////////
	// Current User
	cl.Router.GET(cl.prefix+"/user/fbCurrent", cl.fbCUHandler())

	/////////////////////////////////////////////
	// Update God Mode
	cl.Router.PUT(cl.prefix+"/user/update-god-mode", cl.updateGodModeHandler())

	////////////////////////////////////////////
	// Invitation Group
	iGroup := cl.Router.Group(prefix + "/invitation")

	// New
	iGroup.GET("/new", cl.newInvitationHandler())

	// Create
	iGroup.PUT("/new", cl.createInvitationHandler())

	// Drop
	iGroup.PUT("/drop/:id", cl.dropHandler())

	// Accept
	iGroup.PUT("/accept/:id", cl.acceptHandler())

	// Details
	iGroup.GET("/details/:id", cl.detailsHandler())

	// Abort
	iGroup.PUT("abort/:id", cl.abortHandler())

	/////////////////////////////////////////////
	// Game Group
	gGroup := cl.Router.Group(prefix + "/game")

	// Reset
	gGroup.PUT("reset/:id", cl.resetHandler())

	// Undo
	gGroup.PUT("undo/:id", cl.undoHandler())

	// Redo
	gGroup.PUT("redo/:id", cl.redoHandler())

	// Rollback
	gGroup.PUT("rollback/:id", cl.rollbackHandler())

	// Rollforward
	gGroup.PUT("rollforward/:id", cl.rollforwardHandler())

	// Abandon
	gGroup.PUT("abandon/:id", cl.abandonHandler)

	// Revive
	gGroup.PUT("revive/:id", cl.reviveHandler)

	/////////////////////////////////////////////
	// Message Log
	msg := cl.Router.Group(prefix + "/mlog")

	// Update Read
	msg.PUT("/updateRead/:id", cl.updateReadHandler())

	// Add
	msg.PUT("/add/:id", cl.addMessageHandler())

	return cl
}
