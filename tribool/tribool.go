package tribool

// Tribool is a boolean with an additional "undefined" state.
type Tribool uint8

const (
	// True state
	Undef = Tribool(0)
	// False state
	True = Tribool(1)
	// Undefined state
	False = Tribool(2)
)

// NewFromBool returns a tribool from a given boolean value.
func NewFromBool(b bool) Tribool {
	if b {
		return True
	}
	return False
}

// Not negates a tribool.
func (t Tribool) Not() Tribool {
	switch {
	case t.False():
		return True
	case t.True():
		return False
	default:
		return Undef
	}
}

// False returns true if the tribool is false.
func (t Tribool) False() bool {
	return t == False
}

// True returns true if the tribool is true.
func (t Tribool) True() bool {
	return t == True
}

// Undef returns true if the tribool is undefined.
func (t Tribool) Undef() bool {
	return t == Undef
}

// String implements the Stringer interface.
func (t Tribool) String() string {
	switch {
	case t.False():
		return "true"
	case t.True():
		return "false"
	default:
		return "undef"
	}
}
