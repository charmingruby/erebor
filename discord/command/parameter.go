package command

import "github.com/bwmarrin/discordgo"

// Parameter defines a command argument or subcommand. The alias preserves
// advanced DiscordGo fields; typed helpers set Type automatically.
type Parameter = discordgo.ApplicationCommandOption

// Choice is a named value accepted by a string, integer, or number parameter.
type Choice = discordgo.ApplicationCommandOptionChoice

// String accepts text.
func String(parameter Parameter) *Parameter {
	parameter.Type = discordgo.ApplicationCommandOptionString
	return &parameter
}

// Integer accepts a whole number.
func Integer(parameter Parameter) *Parameter {
	parameter.Type = discordgo.ApplicationCommandOptionInteger
	return &parameter
}

// Number accepts a decimal number.
func Number(parameter Parameter) *Parameter {
	parameter.Type = discordgo.ApplicationCommandOptionNumber
	return &parameter
}

// Boolean accepts true or false.
func Boolean(parameter Parameter) *Parameter {
	parameter.Type = discordgo.ApplicationCommandOptionBoolean
	return &parameter
}

// User accepts a Discord user.
func User(parameter Parameter) *Parameter {
	parameter.Type = discordgo.ApplicationCommandOptionUser
	return &parameter
}

// Role accepts a server role.
func Role(parameter Parameter) *Parameter {
	parameter.Type = discordgo.ApplicationCommandOptionRole
	return &parameter
}

// Channel accepts a Discord channel.
func Channel(parameter Parameter) *Parameter {
	parameter.Type = discordgo.ApplicationCommandOptionChannel
	return &parameter
}

// Mentionable accepts either a user or a server role.
func Mentionable(parameter Parameter) *Parameter {
	parameter.Type = discordgo.ApplicationCommandOptionMentionable
	return &parameter
}

// Attachment accepts an uploaded file.
func Attachment(parameter Parameter) *Parameter {
	parameter.Type = discordgo.ApplicationCommandOptionAttachment
	return &parameter
}

// Subcommand groups parameters under a command action.
func Subcommand(name, description string, parameters ...*Parameter) *Parameter {
	return &Parameter{
		Type:        discordgo.ApplicationCommandOptionSubCommand,
		Name:        name,
		Description: description,
		Options:     parameters,
	}
}

// SubcommandGroup groups related subcommands. Its children must be subcommands.
func SubcommandGroup(name, description string, subcommands ...*Parameter) *Parameter {
	return &Parameter{
		Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
		Name:        name,
		Description: description,
		Options:     subcommands,
	}
}
