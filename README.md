# Slack duty bot
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/iqoption/slack-duty-bot)](https://goreportcard.com/report/github.com/iqoption/slack-duty-bot)

### How usage
1. Create new custom integration `Outgoing WebHooks` (e.g https://{team}.slack.com/apps/manage/custom-integrations)
2. Build for your environment
3. Add a schedule for the attendants 
3. Run with the required parameters

### Build package
```bash
$ env GOOS=linux GOARCH=amd64 go get -v -d ./src/slack-duty-bot
$ env GOOS=linux GOARCH=amd64 go build -o ./bin/slack-duty-bot ./src/slack-duty-bot/main.go
```

### Config
```yaml
token: %some-token%
log: /var/log/slack-duty-bot.log
ids:
  username.one: U11GZZZZZ
  username.two: U11VZZZZZ
duties:
  - [username.one, username.two] # Sunday
  - [username.one] # Monday
  - [username.two] # Tuesday
  - [username.one] # Wednesday
  - [username.two] # Thursday
  - [username.one] # Friday
  - [username.one, username.two] # Saturday
```

### Available arguments
* `--port` - bot http listen port (default: 8003)
* `--config` - path to yml config (default: /var/slack-bot/config.yml)
* `--period` - period in seconds after which the config will be updated (default: 5)
* `--restore` - enable restore previous stable config version process
