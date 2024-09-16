# Telegram Scheduler Bot

A Go-based Telegram bot to automate scheduling and sending messages to a Telegram channel or chat at specified times. Manage scheduled messages directly through the bot with simple commands, store schedules in a JSON file, and execute them daily.

## Features

- **Automated Scheduling**: Schedule messages to be sent at specific times every day.
- **Manage Messages**: Admin can add, view, and delete scheduled messages via simple commands.
- **Persistence**: Messages and schedules are stored in a JSON file to retain configurations across restarts.
- **Telegram Integration**: Leverages the Telegram Bot API for sending messages to a specified channel or chat.

---

## Getting Started

Follow these steps to set up and run the Telegram Scheduler Bot on your machine.

### Prerequisites

- [Go](https://golang.org/doc/install) (version 1.16 or higher)
- A Telegram bot (create one via [BotFather](https://core.telegram.org/bots#botfather))
- A Telegram channel or chat to send messages to

### Installation

1. **Clone the repository**:

    ```bash
    git clone https://github.com/yourusername/telegram-scheduler-bot.git
    cd telegram-scheduler-bot
    ```

2. **Install dependencies**:

    Run the following command to install the required Go modules:

    ```bash
    go mod tidy
    ```

3. **Configure the bot**:

    Open `main.go` and update the following constants with your bot details:

    ```go
    const (
        defaultChatID   = "@yourchannel"    // Telegram channel/chat ID
        defaultBotToken = "your-bot-token"  // Bot Token from BotFather
        adminUserID     = your-admin-id     // Your Telegram user ID (admin-only access)
    )
    ```

---

## Running the Bot

To start the bot, simply run the following command:

```bash
go run main.go
