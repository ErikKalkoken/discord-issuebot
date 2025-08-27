# discord-supportbot

A Discord bot for creating Github issues from messages.

[![Go](https://github.com/ErikKalkoken/discord-supportbot/actions/workflows/go.yml/badge.svg)](https://github.com/ErikKalkoken/discord-supportbot/actions/workflows/go.yml)

## Installation

### Discord App creation

Create a Discord app with the following settings:

- General Information
  - Name: supportbot
  - App Icon: You can find the official icon on the repo in the resources directory.

- Installation
  - Installation Context: Disable "Guild Install" and keep "User Install" enabled

- Bot
  - Message Content Intend: Click to enable (Needed to show the content of a bookmarked message)
  - Token: Click on "Reset" to create a new token and write it down somewhere (or keep the page open)

### Service installation

> [!NOTE]
> This guide uses [supervisor](http://supervisord.org/index.html) for running supportbot as a service. Please make sure it is installed on your system before continuing.

#### Create user

Create a "service" user with disabled login:

```sh
sudo adduser --disabled-login supportbot
```

Switch to the service user and move to the home directory:

```sh
sudo su supportbot
cd ~
```

### Install binary (WIP)

Download and decompress the latest release from the [releases page](https://github.com/ErikKalkoken/supportbot/releases):

```sh
wget https://github.com/ErikKalkoken/supportbot/releases/download/vX.Y.Z/supportbot-X.Y.Z-linux-amd64.tar.gz
tar -xvzf supportbot-X.Y.Z-linux-amd64.tar.gz
```

> [!TIP]
> Please make sure update the URL and filename to the latest version.

#### Supervisor

Download supervisor configuration file:

```sh
wget https://raw.githubusercontent.com/ErikKalkoken/supportbot/main/config/supervisor.conf
```

Setup and configure:

```sh
chmod +x supportbot
touch supportbot.log
```

Add your Discord app ID and bot token to the supervisor.conf file.

Add supportbot to supervisor:

```sh
sudo ln -s /home/supportbot/supervisor.conf /etc/supervisor/conf.d/supportbot.conf
sudo systemctl restart supervisor
```

Restart the supportbot service.

```sh
sudo supervisorctl restart supportbot
```

## Credits

[Contact-us icons created by redempticon - Flaticon](https://www.flaticon.com/free-icons/contact-us)
