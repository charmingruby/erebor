// Package codec converts between DiscordGo payloads and Erebor's public API.
package codec

import (
	"slices"

	"github.com/bwmarrin/discordgo"

	"github.com/charmingruby/erebor/discord/api"
)

// Parameters builds the payload used to publish command arguments.
// Optional bounds retain their pointers; definitions must not be mutated after New.
func Parameters(parameters []api.Parameter) []*discordgo.ApplicationCommandOption {
	out := make([]*discordgo.ApplicationCommandOption, 0, len(parameters))
	for _, p := range parameters {
		option := &discordgo.ApplicationCommandOption{
			Type:                     discordgo.ApplicationCommandOptionType(p.Type),
			Name:                     p.Name,
			Description:              p.Description,
			Required:                 p.Required,
			NameLocalizations:        encodeLocales(p.NameLocalizations),
			DescriptionLocalizations: encodeLocales(p.DescriptionLocalizations),
			MinValue:                 p.MinValue,
			MaxValue:                 p.MaxValue,
			MinLength:                p.MinLength,
			MaxLength:                p.MaxLength,
		}
		for _, kind := range p.ChannelTypes {
			option.ChannelTypes = append(option.ChannelTypes, discordgo.ChannelType(kind))
		}
		for _, choice := range p.Choices {
			option.Choices = append(option.Choices, &discordgo.ApplicationCommandOptionChoice{
				Name:              choice.Name,
				Value:             choice.Value,
				NameLocalizations: encodeLocales(choice.NameLocalizations),
			})
		}
		out = append(out, option)
	}
	return out
}

func encodeLocales(locales map[string]string) map[discordgo.Locale]string {
	out := make(map[discordgo.Locale]string, len(locales))
	for locale, text := range locales {
		out[discordgo.Locale(locale)] = text
	}
	return out
}

// Interaction converts a slash-command invocation into the application contract.
func Interaction(event *discordgo.InteractionCreate, data discordgo.ApplicationCommandInteractionData) api.Interaction {
	user := event.User
	if event.Member != nil {
		user = event.Member.User
	}

	return api.Interaction{
		ID:        event.ID,
		Name:      data.Name,
		Path:      []string{data.Name},
		GuildID:   event.GuildID,
		ChannelID: event.ChannelID,
		User:      decodeUser(user),
		Member:    decodeMember(event.Member),
		Options:   decodeArguments(data.Options),
		Resolved:  decodeResolved(data.Resolved),
	}
}

func decodeArguments(options []*discordgo.ApplicationCommandInteractionDataOption) []*api.Argument {
	out := make([]*api.Argument, 0, len(options))
	for _, option := range options {
		out = append(out, &api.Argument{
			Name:    option.Name,
			Type:    api.ParameterType(option.Type),
			Value:   option.Value,
			Options: decodeArguments(option.Options),
		})
	}
	return out
}

func decodeUser(user *discordgo.User) *api.User {
	if user == nil {
		return nil
	}
	return &api.User{
		ID:            user.ID,
		Username:      user.Username,
		GlobalName:    user.GlobalName,
		Discriminator: user.Discriminator,
		Avatar:        user.Avatar,
		Bot:           user.Bot,
		System:        user.System,
	}
}

func decodeMember(member *discordgo.Member) *api.Member {
	if member == nil {
		return nil
	}
	return &api.Member{
		User:                       decodeUser(member.User),
		GuildID:                    member.GuildID,
		Nick:                       member.Nick,
		Avatar:                     member.Avatar,
		Roles:                      slices.Clone(member.Roles),
		JoinedAt:                   member.JoinedAt,
		PremiumSince:               member.PremiumSince,
		Permissions:                member.Permissions,
		Pending:                    member.Pending,
		Deaf:                       member.Deaf,
		Mute:                       member.Mute,
		CommunicationDisabledUntil: member.CommunicationDisabledUntil,
	}
}

func decodeAttachment(file *discordgo.MessageAttachment) *api.Attachment {
	return &api.Attachment{
		ID:           file.ID,
		URL:          file.URL,
		ProxyURL:     file.ProxyURL,
		Filename:     file.Filename,
		ContentType:  file.ContentType,
		Size:         file.Size,
		Height:       file.Height,
		Width:        file.Width,
		Ephemeral:    file.Ephemeral,
		DurationSecs: file.DurationSecs,
		Waveform:     file.Waveform,
	}
}

func decodeResolved(data *discordgo.ApplicationCommandInteractionDataResolved) *api.Resolved {
	if data == nil {
		return nil
	}
	return &api.Resolved{
		Users:       mapObjects(data.Users, decodeUser),
		Members:     mapObjects(data.Members, decodeMember),
		Attachments: mapObjects(data.Attachments, decodeAttachment),
		Roles: mapObjects(data.Roles, func(role *discordgo.Role) *api.Role {
			return &api.Role{
				ID:          role.ID,
				Name:        role.Name,
				Color:       role.Color,
				Hoist:       role.Hoist,
				Position:    role.Position,
				Permissions: role.Permissions,
				Managed:     role.Managed,
				Mentionable: role.Mentionable,
			}
		}),
		Channels: mapObjects(data.Channels, func(channel *discordgo.Channel) *api.Channel {
			return &api.Channel{
				ID:       channel.ID,
				GuildID:  channel.GuildID,
				Name:     channel.Name,
				Type:     api.ChannelType(channel.Type),
				ParentID: channel.ParentID,
			}
		}),
		Messages: mapObjects(data.Messages, func(message *discordgo.Message) *api.Message {
			out := &api.Message{
				ID:        message.ID,
				ChannelID: message.ChannelID,
				GuildID:   message.GuildID,
				Content:   message.Content,
				Author:    decodeUser(message.Author),
			}
			for _, file := range message.Attachments {
				out.Attachments = append(out.Attachments, decodeAttachment(file))
			}
			return out
		}),
	}
}

func mapObjects[A, B any](source map[string]*A, convert func(*A) *B) map[string]*B {
	if source == nil {
		return nil
	}
	out := make(map[string]*B, len(source))
	for id, value := range source {
		out[id] = convert(value)
	}
	return out
}
