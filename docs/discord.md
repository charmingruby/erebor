# Discord

A reusable bot built on DiscordGo, supporting slash commands and optional
mentions. Application logic stays in callbacks; the package has no dependency on
AI models or application prompts.

## Slash commands

Mention handling is opt-in. Register commands before starting the listener:

```go
bot, err := discord.NewBot(
    os.Getenv("DISCORD_BOT_TOKEN"),
    discord.WithCommandGuildID(os.Getenv("DISCORD_GUILD_ID")), // Empty = global.
    discord.WithErrorMessage("Não consegui executar o comando. Tente novamente."),
)
if err != nil {
    return err
}

err = bot.RegisterCommand(discord.Command{
    Name: "echo",
    Description: "Retorna o texto informado",
    Ephemeral: true,
    Options: []*discordgo.ApplicationCommandOption{
        {
            Name: "text",
            Description: "Texto para retornar",
            Type: discordgo.ApplicationCommandOptionString,
            Required: true,
        },
    },
    Handler: func(ctx context.Context, interaction discord.Interaction) (string, error) {
        for _, option := range interaction.Options {
            if option.Name == "text" {
                return option.StringValue(), nil
            }
        }
        return "", nil
    },
})
if err != nil {
    return err
}

ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
return bot.Listen(ctx)
```

Options use `github.com/bwmarrin/discordgo`: strings, numbers, booleans, users,
roles, channels, attachments, choices, and nested subcommands. The callback gets
`Options`, `Resolved` objects, `User`, `Member`, `ChannelID`, and `GuildID`.
`Member` may be nil in DMs. Autocomplete and context-menu commands are not handled.

- Install with the `bot` and `applications.commands` OAuth scopes. Command-only
  bots do not need Message Content Intent.
- `Listen` creates or updates declared commands individually. It does not delete
  unrelated or removed commands, including commands left in an old scope.
  Registration failures close the connection without rolling back earlier updates.
- Use `WithCommandGuildID` for development; global commands may take time to appear.
- Invocations are deferred before the callback, with a 2.5-second acknowledgment
  timeout. Failed acknowledgments skip execution. `Ephemeral` applies to all chunks.
- Empty results remove the deferred response. Failures use `ErrorMessage`, or
  remove the response when no fallback is configured.
- Register commands before listening; duplicate names are rejected.
  Do not mutate registered option definitions.

## Mentions

To enable mentions, supply `WithHandler`:

```go
discord.WithHandler(func(ctx context.Context, message discord.Message) (string, error) {
    return "Você disse: " + message.Content, nil
})
```

Enable **Message Content Intent** and grant permissions to view channels, send
messages, and read message history. Mentions and replies from guild users are
handled; bots and DMs are ignored. Empty mentions reach the callback. Typing is
refreshed every 8 seconds, and empty answers send nothing.

`ReferencedContent` holds the previous bot reply. Treat it as untrusted data,
not instructions, when building AI prompts.

## Shared behavior

- Both features work together or independently. Supply a mention handler or at
  least one command.
- Defaults: 8 concurrent handlers, 15-second HTTP timeout, 3-minute handler
  timeout, and 1-minute reply timeout. Excess mentions are dropped and logged;
  excess commands receive a private busy response.
- Override these with `WithMaxConcurrentRequests`, `WithRequestTimeout`,
  `WithHandlerTimeout`, and `WithReplyTimeout`. Explicit values must be positive;
  omitted options use defaults. Later options override earlier ones.
- Message whitespace is preserved; only mentions of this bot are removed.
  Command payload rules are validated by Discord during registration.
- Handlers must honor cancellation. Command execution is capped at 14 minutes
  for Discord's token lifetime. Shutdown cancels handlers and waits for them.
- Replies use chunks of at most 2,000 UTF-16 units, preserving Unicode and
  disabling generated mentions. Only the first mention-reply chunk quotes its source.
- Handler failures are logged and optionally send `ErrorMessage`.
- Normal shutdown returns `nil`; an already-canceled context returns its error.
  Concurrent listeners and command registration while listening are rejected.
