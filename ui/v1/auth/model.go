package auth

import (
	"cassette/core/auth"
)

type State int

const (
	NeedsAuth State = iota
	Authenticating
	Authenticated
)

type Model struct {
	authState       State
	auth            *auth.Authenticator
	authFlowUpdates chan string
	err             error
	width           int
	height          int
	copied          bool
}

func NewModel() *Model {
	return &Model{
		authState:       Authenticated,
		auth:            auth.New(),
		authFlowUpdates: make(chan string),
	}
}

func (m *Model) State() State {
	return m.authState
}

func (m *Model) SetState(state State) {
	m.authState = state
}

func (m *Model) Authenticator() *auth.Authenticator {
	return m.auth
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}
