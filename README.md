# Slack duty bot

### Build package
```
env GOOS=linux GOARCH=amd64 go get -v -d ./src/slack-duty-bot
env GOOS=linux GOARCH=amd64 go build -o ./bin/slack-duty-bot ./src/slack-duty-bot/main.go
```

### Config
```
token: %some-token%
log: /var/log/slack-duty-bot.log
ids:
  username.one: U1LGZZZZZ
  username.two: U17VZZZZZ
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