package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Token  string
	Log    string
	Ids    map[string]string
	Duties [][]string
}

type Response struct {
	Text string `json:"text"`
}

var (
	booted  = false
	config  = &Config{}
	file    = flag.String("config", "/var/slack-duty-bot/config.yml", "Full path to yml config")
	port    = flag.String("port", "8003", "HTTP port to listen by bot")
	period  = flag.Int("period", 5, "Period in seconds after which the config will be updated")
	restore = flag.Bool("restore", false, "Enable restore previous stable config version process")
)

func main() {
	flag.Parse()
	go updateConfig()
	http.HandleFunc("/", rootHandler)
	http.ListenAndServe(fmt.Sprintf(":%s", *port), nil)
}

func readConfig() (*Config, error) {
	content, err := ioutil.ReadFile(*file)
	if err != nil {
		return nil, err
	}
	current := &Config{}
	if err := yaml.Unmarshal(content, current); err != nil {
		return nil, err
	}
	if current.Token == "" {
		return nil, errors.New("Config token is empty")
	}
	if len(current.Duties) < 7 {
		return nil, errors.New(fmt.Sprintf("Invalid number (%d) of messages in config", len(config.Duties)))
	}
	if current.Log == "" {
		return nil, errors.New("Config log is empty")
	}
	return current, nil
}

func updateConfig() {
	for {
		current, err := readConfig()
		if err != nil {
			if !*restore || booted == false {
				log.Panicln(fmt.Sprintf("Failed to init config. Error: %s", err.Error()))
			}
			log.Printf("An error occured during update config (\"%s\"). Restore previous version.", err.Error())
			current = config
		}
		config = current
		if booted == false {
			booted = true
		}
		time.Sleep(time.Second * time.Duration(*period))
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if result, err := validate(r.Body); !result {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := getResponse()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func validate(context io.ReadCloser) (result bool, err error) {
	data, err := ioutil.ReadAll(context)
	if err != nil {
		return false, err
	}

	body := string(data)
	if !strings.Contains(body, config.Token) {
		return false, errors.New("Bad token")
	}

	descriptor, err := os.OpenFile(config.Log, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755)
	if err != nil {
		return false, err
	}

	defer descriptor.Close()
	descriptor.WriteString(fmt.Sprintf("[%s] Incoming request body: %s", time.Now(), body))

	return true, err
}

func getResponse() (Response, error) {
	duties, err := arrayMap(config.Duties[time.Now().Weekday()], func(user string) (string, error) {
		id, ok := config.Ids[user]
		if !ok {
			return "", errors.New(fmt.Sprintf("Unknown user \"%s\" called from config", user))
		}
		return fmt.Sprintf("<@%s|%s>", id, user), nil
	})
	response := Response{}
	if err != nil {
		return response, err
	}
	response.Text = strings.Join(duties, " ")
	return response, nil
}

func arrayMap(array []string, callback func(string) (string, error)) ([]string, error) {
	arrayCopy := make([]string, len(array))
	for index, value := range array {
		value, err := callback(value)
		if err != nil {
			return nil, err
		}
		arrayCopy[index] = value
	}
	return arrayCopy, nil
}
