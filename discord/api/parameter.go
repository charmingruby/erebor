package api

// ParameterType identifies a Discord slash-command argument.
type ParameterType uint8

// Discord slash-command parameter kinds.
const (
	ParameterSubcommand ParameterType = iota + 1
	ParameterGroup
	ParameterString
	ParameterInteger
	ParameterBoolean
	ParameterUser
	ParameterChannel
	ParameterRole
	ParameterMentionable
	ParameterNumber
	ParameterAttachment
)

// Parameter defines a command argument. Discord validates names, limits and
// choices when commands are published. Do not mutate parameters after bot creation.
type Parameter struct {
	NameLocalizations        map[string]string
	DescriptionLocalizations map[string]string
	MinValue                 *float64
	MinLength                *int
	Name                     string
	Description              string
	ChannelTypes             []ChannelType
	Choices                  []Choice
	MaxValue                 float64
	MaxLength                int
	Type                     ParameterType
	Required                 bool
}

// Choice is a named value accepted by a string, integer, or number parameter.
type Choice struct {
	Value             any
	NameLocalizations map[string]string
	Name              string
}
