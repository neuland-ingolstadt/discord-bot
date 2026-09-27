# Onboarding Ticket Bot

Go Discord bot that opens a private onboarding ticket when someone receives the **Interessent** role. Staff (`@Vorstand`, `@Management`) can close the ticket with a confirmation button.

Licensed under the [European Union Public Licence v1.2 (EUPL-1.2)](LICENSE).

## Container image

On every push to `main`, CI publishes to GHCR with a unique run number:

- `ghcr.io/neuland-ingolstadt/discord-bot:main`
- `ghcr.io/neuland-ingolstadt/discord-bot:main-<run_number>`

Example: `ghcr.io/neuland-ingolstadt/discord-bot:main-42`

## Features

- Creates `ticket-XXXX` under the **Onboarding** category
- Visible/writable only by the member plus Vorstand and Management
- Posts a welcome message with a **Ticket schließen** button
- Staff-only close flow with ephemeral confirm/cancel; confirm deletes the channel
- Slash command `/roles` — lists all server roles as `` `Name` — `ID` ``
- Slash command `/connect` — ephemeral link to Neuland Connect account linking

## Requirements

- Go 1.21+
- A Discord application/bot with privileged **Server Members Intent** enabled
- Bot invite with **applications.commands** scope (needed for slash commands)

**Note:** Leave the Discord Developer Portal **Interactions Endpoint URL** empty so slash commands are handled over the gateway by this bot (not by Neuland Connect).

## Discord setup

1. Create an application at [Discord Developer Portal](https://discord.com/developers/applications) → Bot → Reset Token.
2. Under **Bot → Privileged Gateway Intents**, enable **Server Members Intent**.
3. Invite the bot with permissions:
   - Manage Channels
   - View Channels
   - Send Messages
   - Read Message History
4. On the server, create a category named **Onboarding** (or reuse an existing one).
5. Copy IDs (Developer Mode → right-click → Copy ID):
   - Server → `GUILD_ID`
   - Onboarding category → `ONBOARDING_CATEGORY_ID`
   - Roles Interessent, Vorstand, Management → the three role env vars

## Configuration

```bash
cp .env.example .env
# fill in all values
```

| Variable | Description |
|---|---|
| `DISCORD_TOKEN` | Bot token |
| `GUILD_ID` | Target guild |
| `ONBOARDING_CATEGORY_ID` | Parent category for tickets |
| `INTERESSENT_ROLE_ID` | Role that triggers ticket creation |
| `VORSTAND_ROLE_ID` | Staff role (view + close) |
| `MANAGEMENT_ROLE_ID` | Staff role (view + close) |
| `CONNECT_URL` | Optional. Link for `/connect` (default `https://connect.neuland.ing/connect`) |

## Run

```bash
go run ./cmd/bot
```

Or build:

```bash
make build
./bin/bot
```

Docker:

```bash
make docker-build
```

## Flow

1. Staff assigns the Interessent role to a member.
2. Bot creates a private channel and posts the welcome message.
3. Staff clicks **Ticket schließen** → ephemeral confirmation → **Bestätigen** deletes the channel.
