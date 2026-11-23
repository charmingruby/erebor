package api

// Resolved contains objects referenced by argument IDs. Maps may be nil when no
// arguments of that kind were supplied.
type Resolved struct {
	Users       map[string]*User
	Members     map[string]*Member
	Roles       map[string]*Role
	Channels    map[string]*Channel
	Attachments map[string]*Attachment
	Messages    map[string]*Message
}

// Argument is a supplied command parameter. Entity values are their Discord IDs.
// Numeric values use float64, matching Discord's JSON payload.
type Argument struct {
	Value   any
	Name    string
	Options []*Argument
	Type    ParameterType
}

// StringValue returns text or an entity ID, or "" for a different value type.
func (a Argument) StringValue() string {
	value, _ := a.Value.(string)
	return value
}

// IntValue returns an integer argument, or zero for a different value type.
func (a Argument) IntValue() int64 {
	return int64(a.FloatValue())
}

// FloatValue returns a numeric argument, or zero for a different value type.
func (a Argument) FloatValue() float64 {
	value, _ := a.Value.(float64)
	return value
}

// BoolValue returns a boolean argument, or false for a different value type.
func (a Argument) BoolValue() bool {
	value, _ := a.Value.(bool)
	return value
}

// GetOption returns a supplied argument, or nil when absent.
func (i Interaction) GetOption(name string) *Argument {
	for _, option := range i.Options {
		if option.Name == name {
			return option
		}
	}
	return nil
}

// String returns text or an entity ID. An absent argument returns "".
func (i Interaction) String(name string) string {
	if option := i.GetOption(name); option != nil {
		return option.StringValue()
	}
	return ""
}

// Integer returns an integer argument. An absent argument returns zero.
func (i Interaction) Integer(name string) int64 {
	if option := i.GetOption(name); option != nil {
		return option.IntValue()
	}
	return 0
}

// Number returns a decimal argument. An absent argument returns zero.
func (i Interaction) Number(name string) float64 {
	if option := i.GetOption(name); option != nil {
		return option.FloatValue()
	}
	return 0
}

// Boolean returns a boolean argument. An absent argument returns false.
func (i Interaction) Boolean(name string) bool {
	if option := i.GetOption(name); option != nil {
		return option.BoolValue()
	}
	return false
}
