package protocol

// State represents a Minecraft protocol state.
type State int

const (
	StateHandshake     State = -1
	StateStatus        State = 1
	StateLogin         State = 2
	StatePlay          State = 3
	StateConfiguration State = 4 // 1.20.5+
)

func (s State) String() string {
	switch s {
	case StateHandshake:
		return "handshake"
	case StateStatus:
		return "status"
	case StateLogin:
		return "login"
	case StatePlay:
		return "play"
	case StateConfiguration:
		return "configuration"
	default:
		return "unknown"
	}
}
