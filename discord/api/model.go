package api

import "time"

// User identifies a Discord account.
type User struct {
	ID            string
	Username      string
	GlobalName    string
	Discriminator string
	Avatar        string
	Bot           bool
	System        bool
}

// Member describes a user's membership in a guild. User may be nil in resolved
// data; the corresponding account is available in Resolved.Users by ID.
type Member struct {
	JoinedAt                   time.Time
	User                       *User
	PremiumSince               *time.Time
	CommunicationDisabledUntil *time.Time
	GuildID                    string
	Nick                       string
	Avatar                     string
	Roles                      []string
	Permissions                int64
	Pending                    bool
	Deaf                       bool
	Mute                       bool
}

// Role describes a guild role.
type Role struct {
	ID          string
	Name        string
	Color       int
	Position    int
	Permissions int64
	Hoist       bool
	Managed     bool
	Mentionable bool
}

// ChannelType identifies a Discord channel kind.
type ChannelType int

// Basic Discord channel kinds.
const (
	GuildText ChannelType = iota
	DM
	GuildVoice
	GroupDM
	GuildCategory
	GuildAnnouncement
)

// Thread and extended guild channel kinds.
const (
	AnnouncementThread ChannelType = 10
	PublicThread       ChannelType = 11
	PrivateThread      ChannelType = 12
	GuildStageVoice    ChannelType = 13
	GuildDirectory     ChannelType = 14
	GuildForum         ChannelType = 15
	GuildMedia         ChannelType = 16
)

// Channel contains the channel data resolved by Discord for a command.
type Channel struct {
	ID       string
	GuildID  string
	Name     string
	ParentID string
	Type     ChannelType
}

// Attachment describes an uploaded file.
type Attachment struct {
	ID           string
	URL          string
	ProxyURL     string
	Filename     string
	ContentType  string
	Waveform     string
	Size         int
	Height       int
	Width        int
	DurationSecs float64
	Ephemeral    bool
}

// Message contains a message resolved by an interaction.
type Message struct {
	ID          string
	ChannelID   string
	GuildID     string
	Content     string
	Author      *User
	Attachments []*Attachment
}
