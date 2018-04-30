# Slack duty bot
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/iqoption/slack-duty-bot)](https://goreportcard.com/report/github.com/iqoption/slack-duty-bot)

### Usage
1. Create new custom integration `Bots` (e.g https://{team}.slack.com/apps/manage/custom-integrations)
2. Add bot to channels you want to listen
2. Build for your environment
3. Prepare config.yaml with duties list
3. Run with the required parameters

```bash
SDB_SLACK_TOKEN=your-token-here ./slack-duty-bot \
    --slack.keyword keyword-1 \
    --slack.keyword keyword-2 \
    --slack.group.id your-group-id \
    --slack.group.name your-group-name
```

### Build package
Build
```bash
go get -u github.com/golang/dep/cmd/dep
dep ensure
env GOOS=linux GOARCH=amd64 go build -v
```
Build via makefile
```bash
make BUILD_OS=linux BUILD_ARCH=amd64
```
Build in docker
```bash
docker run --rm -v $(pwd):/go/src/slack-duty-bot -w /go/src/slack-duty-bot golang:1.10 make BUILD_OS=linux BUILD_ARCH=amd64
```

### Configuration

#### Configuration flags, environment variables
Environment variables are prefixed with `SDB_` and *MUST* be uppercase with `_` delimiter
Available variables:
* `SDB_SLACK_TOKEN`
* `SDB_SLACK_GROUP_ID`
* `SDB_SLACK_GROUP_NAME`
* `SDB_SLACK_THREADS`

Every environment variable can be overwritten by startup flags
Available flags:
* `--config.path` - path to yml config (default: . and $HOME/.slack-duty-bot)
* `--slack.token` - Slack API client token
* `--slack.keyword` - Case insensitive keywords slice to search in message text, can be set multiple times (default: [])
* `--slack.group.name` - Slack user group name, to mention in channel if duty list is empty
* `--slack.group.id` - Slack user group ID, to mention in channel if duty list is empty
* `--slack.threads` - Use threads as reply target or push message direct to channel (default: true) 

You can get IDS from api or just use [testing page](https://api.slack.com/methods/usergroups.list/test)

#### Configuration file
Configuration file *MUST* contain `duties` key with *7* slices of Slack user names
```yaml
duties:
  - [username.one, username.two] # Sunday
  - [username.one] # Monday
  - [username.two] # Tuesday
  - [username.one] # Wednesday
  - [username.two] # Thursday
  - [username.one] # Friday
  - [username.one, username.two] # Saturday
```

#### Configuration priority
* Flags
* Environment variables
* Config file
