package command

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

// Definition describes a slash command. Do not mutate its parameters after registration.
type Definition struct {
	Name        string
	Description string
	Handler     Handler
	Parameters  []*Parameter
	// OnlyVisibleToCaller hides every response chunk from other users.
	OnlyVisibleToCaller bool
}

// Handler processes a command. It may run concurrently and must honor context
// cancellation. Empty output removes the deferred response.
type Handler func(context.Context, Interaction) (string, error)

// Interaction contains command arguments and the invoking user and channel.
// Options preserve subcommand hierarchy; Resolved holds referenced objects.
// Member may be nil in direct messages.
type Interaction struct {
	ID        string
	Name      string
	ChannelID string
	GuildID   string
	User      *discordgo.User
	Member    *discordgo.Member
	Resolved  *discordgo.ApplicationCommandInteractionDataResolved
	Options   []*discordgo.ApplicationCommandInteractionDataOption
}
