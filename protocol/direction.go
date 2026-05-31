package protocol

// Direction specifies the packet direction.
type Direction int

const (
	Serverbound Direction = iota
	Clientbound
)

func (d Direction) String() string {
	switch d {
	case Serverbound:
		return "serverbound"
	case Clientbound:
		return "clientbound"
	default:
		return "unknown"
	}
}
