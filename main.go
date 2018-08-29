package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nlopes/slack"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

const (
	incomingErrorRetry = 500
)

func init() {
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.slack-duty-bot")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("SDB")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.BindEnv("slack.token")
	viper.BindEnv("slack.group.id")
	viper.BindEnv("slack.group.name")
	viper.BindEnv("slack.threads")
	viper.AutomaticEnv()

	pflag.String("logger.level", "debug", "Log level")
	pflag.String("config.path", "", "Config path")
	pflag.String("slack.token", "", "Slack API client token config")
	pflag.String("slack.group.name", "", "Slack group ID for calling in fallback mode")
	pflag.String("slack.group.id", "", "Slack group name for calling in fallback mode")
	pflag.StringSlice("slack.keyword", []string{}, "Slack keywords to lister")
	pflag.Bool("slack.threads", true, "Usage of Slack threads to reply on messages")

	viper.BindPFlags(pflag.CommandLine)

	log.SetFormatter(&log.TextFormatter{DisableColors: true})
	log.SetLevel(log.DebugLevel)
}

func main() {
	pflag.Parse()

	if path := viper.GetString("config.path"); path != "" {
		viper.AddConfigPath(path)
	}
	viper.ReadInConfig()
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		log.Debugln("Reloading configuration")
	})

	if level, err := log.ParseLevel(viper.GetString("logger.level")); err != nil && level != log.DebugLevel {
		log.SetLevel(level)
	}

	if err := validateArguments(); err != nil {
		log.Fatalf("Validation arguments error: %+v", err)
	}

	go watchConfig()

	var (
		slackRTM = slack.New(viper.GetString("slack.token")).NewRTM()
	)

	log.Infoln("Send request for RTM connection")
	go slackRTM.ManageConnection()

	var incomingErrorCount = 0
	for packet := range slackRTM.IncomingEvents {
		switch event := packet.Data.(type) {
		case *slack.ConnectedEvent:
			log.Infoln("RTM connection established")

		case *slack.InvalidAuthEvent:
			slackRTM.Disconnect()
			log.Fatalln("Could not authenticate, invalid Slack token passed, terminate")

		case *slack.IncomingEventError:
			incomingErrorCount++
			log.Errorf("RTM incoming error: %+v", event.Error())
			if incomingErrorCount >= incomingErrorRetry {
				slackRTM.Disconnect()
				log.Fatalf("Reached error reconnect limit %d on %s type error, terminate", incomingErrorRetry, packet.Type)
			}

		case *slack.MessageEvent:
			log.Println("Incoming message event")
			log.Debugf("Message event: %+v", event)
			if err := handleMessageEvent(slackRTM, event); err != nil {
				log.Errorf("Handle message event error: %v", err)
			}
		}
	}
}

func watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to initialize config watcher: %+v", err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Debugln("Reloading configuration")
					viper.ReadInConfig()
				}
			case err := <-watcher.Errors:
				log.Debugf("Watcher error: %+v", err)
			}
		}
	}()

	err = watcher.Add(viper.GetString("config.path"))
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func validateArguments() error {
	if viper.GetString("slack.token") == "" {
		return fmt.Errorf("parameter slack.token is required")
	}
	if len(viper.GetStringSlice("slack.keyword")) == 0 {
		return fmt.Errorf("parameter slack.keyword is required")
	}
	return nil
}

func handleMessageEvent(rtm *slack.RTM, event *slack.MessageEvent) error {
	if err := checkMessageEvent(event); err != nil {
		log.Debugf("Check message error: %+v", err)
		return nil
	}

	log.Infof("Incoming message text: %s", event.Text)

	// collection user ids for make duties list
	var userIds = make(map[string]string, 0)
	users, err := rtm.Client.GetUsers()
	if err != nil {
		log.Errorf("Failed to get users list from Slack API: %v", err)
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
			log.Errorf("Failed to get user id by username %s", username)
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
	log.Debugf("Outgoing message: %+v", outgoing)
	rtm.SendMessage(outgoing)
	return nil
}

func checkMessageEvent(event *slack.MessageEvent) error {
	if event.Topic != "" {
		return fmt.Errorf("inocming message about topic change")
	}
	// check text
	if event.Text == "" {
		return fmt.Errorf("incoming message with empty text")
	}
	// check keywords
	contains := any(viper.GetStringSlice("slack.keyword"), func(keyword string) bool {
		return strings.Contains(strings.ToLower(event.Text), strings.ToLower(keyword))
	})
	if contains == false {
		return fmt.Errorf("incoming message text does not contain any suitable keywords")
	}
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
