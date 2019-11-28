package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/aandryashin/reloader"
	"github.com/nlopes/slack"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	incomingErrorRetry = 500
)

type slackLogAdapter struct {
	entry *logrus.Entry
}

func (a *slackLogAdapter) Write(b []byte) (int, error) {
	n := len(b)
	if n > 0 && b[n-1] == '\n' {
		b = b[:n-1]
	}
	a.entry.Info(string(b))
	return n, nil
}

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{DisableColors: true})
	logrus.SetLevel(logrus.InfoLevel)

	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.slack-duty-bot")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("SDB")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	_ = viper.BindEnv("logger.level")
	_ = viper.BindEnv("slack.token")
	_ = viper.BindEnv("slack.group.id")
	_ = viper.BindEnv("slack.group.name")
	_ = viper.BindEnv("slack.threads")
	viper.AutomaticEnv()

	pflag.String("logger.level", "info", "Log level")
	pflag.String("config.path", "", "Config path")
	pflag.String("slack.token", "", "Slack API client token config")
	pflag.String("slack.group.name", "", "Slack group ID for calling in fallback mode")
	pflag.String("slack.group.id", "", "Slack group name for calling in fallback mode")
	pflag.StringSlice("slack.keyword", []string{}, "Slack keywords to lister")
	pflag.Bool("slack.threads", true, "Usage of Slack threads to reply on messages")

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		logrus.Fatalf("Failed to bind command line arguments: %s", err.Error())
	}
}

func main() {
	pflag.Parse()

	if path := viper.GetString("config.path"); path != "" {
		viper.AddConfigPath(path)
	}

	if err := viper.ReadInConfig(); err != nil {
		logrus.Fatalf("Failed to read config file: %s", err.Error())
	}

	err := reloader.Watch(filepath.Dir(viper.ConfigFileUsed()), watcherFunc, 5*time.Second)
	if err != nil {
		logrus.Fatalf("Failed to init config watcher: %s", err.Error())
	}

	if level, err := logrus.ParseLevel(viper.GetString("logger.level")); err == nil {
		logrus.SetLevel(level)
	}

	if err := validateArguments(); err != nil {
		logrus.Fatalf("Validation arguments error: %s", err.Error())
	}

	var slackOptions = make([]slack.Option, 0)
	if logrus.GetLevel() == logrus.DebugLevel {
		slackOptions = append(slackOptions, slack.OptionDebug(true))
		slackOptions = append(slackOptions, slack.OptionLog(log.New(&slackLogAdapter{
			entry: logrus.StandardLogger().WithField("CONTEXT", "slack"),
		}, "", 0)))
	}

	var (
		client   = slack.New(viper.GetString("slack.token"), slackOptions...)
		slackRTM = client.NewRTM()
	)

	logrus.Infoln("Send request for RTM connection")
	go slackRTM.ManageConnection()

	var incomingErrorCount = 0
	for packet := range slackRTM.IncomingEvents {
		logrus.Debugf("Incoming event packet: %+v:", packet)

		switch event := packet.Data.(type) {
		case *slack.ConnectedEvent:
			logrus.Infoln("RTM connection established")
			break

		case *slack.InvalidAuthEvent:
			if err := slackRTM.Disconnect(); err != nil {
				logrus.Errorln("Failed to disconnect from RTM")
			}
			logrus.Fatalln("Could not authenticate, invalid Slack token passed, terminate")
			break

		case *slack.IncomingEventError:
			incomingErrorCount++
			logrus.Errorf("RTM incoming error: %s", event.Error())
			if incomingErrorCount >= incomingErrorRetry {
				if err := slackRTM.Disconnect(); err != nil {
					logrus.Errorln("Failed to disconnect from RTM")
				}
				logrus.Fatalf("Reached error reconnect limit %d on %s type error, terminate", incomingErrorRetry, packet.Type)
			}
			break

		case *slack.MessageEvent:
			if err := handleMessageEvent(slackRTM, event); err != nil {
				logrus.Errorf("Handle message event error: %s", err.Error())
			}
			break
		}
	}
}

func watcherFunc() {
	if err := viper.ReadInConfig(); err != nil {
		logrus.Errorf("Failed to update config on fs event: %s", err.Error())
	}
	logrus.Infoln("Config updated on fs event")
}

func validateArguments() error {
	if viper.GetString("slack.token") == "" {
		return fmt.Errorf("parameter slack.token is required")
	}
	if len(viper.GetStringSlice("slack.keyword")) == 0 {
		return fmt.Errorf("parameter slack.keyword is required")
	}
	var (
		duties   = parseDutiesList()
		index    = getCurrentDayIndex()
		cfgGroup = viper.GetString("slack.group.id") != "" && viper.GetString("slack.group.name") != ""
		length   = len(duties)
	)
	switch length > 0 {
	case true:
		if length < 7 {
			return fmt.Errorf("duties list is not empty, but indexes count %d is less than 7 (weekdays)", length)
		}
		if len(duties[index]) == 0 && !cfgGroup {
			return fmt.Errorf("empty duties list for current day (%d) index and slack.group info does not exist in config", index)
		}
		break
	case false:
		if !cfgGroup {
			return fmt.Errorf("empty duties list in config and slack.group info does not exist in config")
		}
		break
	}
	return nil
}

func parseDutiesList() [][]string {
	var (
		config = struct {
			Duties [][]string // we need this hack cause viper cannot resolve slice of slice
		}{}
	)
	_ = viper.Unmarshal(&config)
	return config.Duties
}

func getCurrentDayIndex() int {
	return int(time.Now().Weekday())
}

func handleMessageEvent(rtm *slack.RTM, event *slack.MessageEvent) error {
	if err := checkMessageEvent(event); err != nil {
		logrus.Debugf("Incoming message check error: %s", err.Error())
		return nil
	}

	logrus.Infof("Incoming message text: %s", event.Text)

	// collection user ids for make duties list
	var userIds = make(map[string]string, 0)
	users, err := rtm.Client.GetUsers()
	if err != nil {
		logrus.Errorf("Failed to get users list from Slack API: %s", err.Error())
	}
	if users != nil {
		for _, user := range users {
			userIds[user.Name] = user.ID
		}
	}
	var (
		duties []string
		cfg    = parseDutiesList()
		index  = getCurrentDayIndex()
	)
	if len(cfg) > index {
		for _, username := range cfg[index] {
			userId, ok := userIds[username]
			if !ok {
				logrus.Errorf("Failed to get user id by username %s", username)
			}
			duties = append(duties, fmt.Sprintf("<@%s|%s>", userId, username))
		}
	}
	if len(duties) == 0 && viper.GetString("slack.group.id") != "" && viper.GetString("slack.group.name") != "" {
		duties = append(duties, fmt.Sprintf("<!subteam^%s|@%s>", viper.GetString("slack.group.id"), viper.GetString("slack.group.name")))
	}
	if len(duties) == 0 {
		return fmt.Errorf("failed to collect duties list for incoming message")
	}
	logrus.Debugf("Final duties list for call: %+v", duties)

	//send message
	var rtmOptions = make([]slack.RTMsgOption, 0)
	if viper.GetBool("slack.threads") == true {
		var (
			optionTS = event.Timestamp
		)
		// message already in timestamp
		if event.ThreadTimestamp != `` {
			optionTS = event.ThreadTimestamp
		}
		rtmOptions = append(rtmOptions, slack.RTMsgOptionTS(optionTS))
	}
	var outgoingMessage = rtm.NewOutgoingMessage(strings.Join(duties, ", "), event.Channel, rtmOptions...)
	logrus.Debugf("Outgoing message: %+v", outgoingMessage)

	rtm.SendMessage(outgoingMessage)
	return nil
}

func checkMessageEvent(event *slack.MessageEvent) error {
	// skip topic messages
	if event.Topic != "" {
		return fmt.Errorf("the incoming message about topic change")
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
