package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/nlopes/slack"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"strings"
	"time"
)

const (
	incomingErrorRetry = 500
)

func init() {
	pflag.String("config.path", "", "Config path")
	pflag.String("slack.token", "", "Slack API client token config")
	// We need ID and name only because bot users can't read user groups info via api
	pflag.String("slack.group.name", "", "Slack group ID for calling in fallback mode")
	pflag.String("slack.group.id", "", "Slack group name for calling in fallback mode")
	pflag.StringSlice("slack.keyword", []string{}, "Slack keywords to lister")
	pflag.Bool("slack.threads", true, "Usage of Slack threads to reply on messages")
	viper.BindPFlags(pflag.CommandLine)

	viper.AutomaticEnv()
	viper.SetEnvPrefix("SDB")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", ".", "_", "_"))
	viper.BindEnv("slack_token")
	viper.BindEnv("slack_group_id")
	viper.BindEnv("slack_group_name")
	viper.BindEnv("slack_threads")

	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.slack-duty-bot")
	viper.AddConfigPath(".")

	log.SetFormatter(&log.TextFormatter{DisableColors: true})
	log.SetLevel(log.DebugLevel)
}

func main() {
	pflag.Parse()

	viper.AddConfigPath(viper.GetString("config.path"))
	viper.ReadInConfig()
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Infoln("Config file was changed")
	})

	if viper.GetString("slack.token") == "" {
		log.Fatalln("Parameter slack.token is required")
	}
	if len(viper.GetStringSlice("slack.keyword")) == 0 {
		log.Fatalln("Parameter slack.keyword is required")
	}
	if viper.GetString("config.path") != "" {
		viper.AddConfigPath(viper.GetString("config.path"))
	}

	var (
		client = slack.New(viper.GetString("slack.token"))
		rtm    = client.NewRTM()
	)

	log.Infoln("Send request for RTM connection")
	go rtm.ManageConnection()

	var incomingErrorCount = 0
	for packet := range rtm.IncomingEvents {
		log.Debugf("Incoming event with type %s", packet.Type)

		switch event := packet.Data.(type) {
		case *slack.ConnectedEvent:
			log.Infoln("RTM connection established")

		case *slack.InvalidAuthEvent:
			rtm.Disconnect()
			log.Fatalf("Could not authenticate, invalid Slack token passed, terminate")

		case *slack.IncomingEventError:
			incomingErrorCount++
			log.Warningf("RTM incoming error: %+v", event.Error())
			if incomingErrorCount >= incomingErrorRetry {
				rtm.Disconnect()
				log.Fatalf("Reached error reconnect limit %d on %s type error, terminate", incomingErrorRetry, packet.Type)
			}

		case *slack.MessageEvent:
			log.Printf("Incoming message event")
			if err := handleMessageEvent(rtm, event); err != nil {
				log.Warningf("Handle message event error: %v", err)
			}
		}
	}
}

func handleMessageEvent(rtm *slack.RTM, event *slack.MessageEvent) error {
	// check text
	if event.Text == "" {
		return fmt.Errorf("incoming message with empty text")
	}
	// check keywords
	var keywords = viper.GetStringSlice("slack.keyword")
	contains := any(keywords, func(keyword string) bool {
		return strings.Contains(strings.ToLower(event.Text), strings.ToLower(keyword))
	})
	if contains == false {
		return fmt.Errorf("incoming message text '%s' does not contain any suitable keywords (%s)", event.Text, strings.Join(keywords, ", "))
	}
	log.Infof("Incoming message text: %s", event.Text)
	// collection user ids for make duties list
	var userIds = make(map[string]string, 0)
	users, err := rtm.Client.GetUsers()
	if err != nil {
		log.Warningf("Failed to get users list from Slack API: %v", err)
	}
	if users != nil {
		for _, user := range users {
			userIds[user.Name] = user.ID
		}
	}
	var (
		config = struct {
			Duties [][]string // we need this hack cause viper cannot resolve slice of slice
		}{}
		duties []string
	)
	viper.Unmarshal(&config)
	for _, username := range config.Duties[int(time.Now().Weekday())] {
		userId, ok := userIds[username]
		if !ok {
			log.Warningf("Failed to get user id by username %s", username)
		}
		duties = append(duties, fmt.Sprintf("<@%s|%s>", userId, username))
	}
	if len(duties) == 0 && viper.GetString("slack.group.id") != "" && viper.GetString("slack.group.name") != "" {
		duties = append(duties, fmt.Sprintf("<!subteam^%s|@%s>", viper.GetString("slack.group.id"), viper.GetString("slack.group.name")))
	}
	// send message
	var outgoing = rtm.NewOutgoingMessage(strings.Join(duties, ", "), event.Channel)
	if viper.GetBool("slack.threads") == true {
		outgoing.ThreadTimestamp = event.Timestamp
	}
	log.Infof("Outgoing message: %+v", outgoing)
	rtm.SendMessage(outgoing)
	return nil
}

func any(vs []string, f func(string) bool) bool {
	for _, v := range vs {
		if f(v) {
			return true
		}
	}
	return false
}
