# discord-issuebot

A bot for creating GitHub or GitLab issues from messages on Discord.

[![Go](https://github.com/ErikKalkoken/discord-issuebot/actions/workflows/go.yml/badge.svg)](https://github.com/ErikKalkoken/discord-issuebot/actions/workflows/go.yml)

## Installation

### Discord App creation

Create a Discord app with the following settings:

- General Information
  - Name: Issue Bot
  - App Icon: You can find the official icon on the repo in the resources directory.

- Installation
  - Installation Context: Disable "Guild Install" and keep "User Install" enabled

- Bot
  - Message Content Intend: Click to enable (Needed to show the content of a bookmarked message)
  - Token: Click on "Reset" to create a new token and write it down somewhere (or keep the page open)

### Service installation

> [!NOTE]
> This guide uses [supervisor](http://supervisord.org/index.html) for running issuebot as a service. Please make sure it is installed on your system before continuing.

#### Create user

Create a "service" user with disabled login:

```sh
sudo adduser --disabled-login issuebot
```

Switch to the service user and move to the home directory:

```sh
sudo su issuebot
cd ~
```

### Install binary (WIP)

Download and decompress the latest release from the [releases page](https://github.com/ErikKalkoken/issuebot/releases):

```sh
wget https://github.com/ErikKalkoken/issuebot/releases/download/vX.Y.Z/issuebot-X.Y.Z-linux-amd64.tar.gz
tar -xvzf issuebot-X.Y.Z-linux-amd64.tar.gz
```

> [!TIP]
> Please make sure update the URL and filename to the latest version.

#### Supervisor

Download supervisor configuration file:

```sh
wget https://raw.githubusercontent.com/ErikKalkoken/issuebot/main/config/supervisor.conf
```

Setup and configure:

```sh
chmod +x issuebot
touch issuebot.log
```

Add your Discord app ID and bot token to the supervisor.conf file.

Add issuebot to supervisor:

```sh
sudo ln -s /home/issuebot/supervisor.conf /etc/supervisor/conf.d/issuebot.conf
sudo systemctl restart supervisor
```

Restart the issuebot service.

```sh
sudo supervisorctl restart issuebot
```

## Credits

[Contact-us icons created by redempticon - Flaticon](https://www.flaticon.com/free-icons/contact-us)
