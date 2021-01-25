package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

var api = slack.New(os.Getenv("SLACK_COPROXY_SLACK_BOT_TOKEN"))

func main() {

	http.HandleFunc("/slack/event", func(w http.ResponseWriter, r *http.Request) {
		bufMaster := bytes.NewBuffer(nil)
		requestBody := io.TeeReader(r.Body, bufMaster)
		body, err := ioutil.ReadAll(requestBody)
		if err != nil {
			log.Println("ioutil.ReadAll(r.Body)")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// シグネチャ計算
		signingSecret := os.Getenv("SLACK_COPROXY_SLACK_SIGNING_SECRET")
		sv, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := sv.Write(body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := sv.Ensure(); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		//*/

		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			log.Println("slackevents.ParseEvent")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// verificationはslack-coproxyが受け持つ
		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				log.Println("json.unmarshal")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))
		}
		if eventsAPIEvent.Type == slackevents.CallbackEvent {
			innerEvent := eventsAPIEvent.InnerEvent
			var httpClient = &http.Client{}

			// ここ本命
			switch innerEvent.Data.(type) {
			case *slackevents.MessageEvent:
				log.Println("slack-coproxy: message_event")
				req, err := http.NewRequest("POST", "http://localhost:5001/slack/event", bytes.NewBuffer([]byte(body)))
				if err != nil {
					log.Println("http.NewRequest: ", err.Error())
				}
				req.Header = r.Header
				log.Println(httpClient.Do(req))
			case *slackevents.AppMentionEvent:
				log.Println("slack-coproxy: app_mention_event")
				req, err := http.NewRequest("POST", "http://localhost:5001/slack/event", bytes.NewBuffer([]byte(body)))
				if err != nil {
					log.Println("http.NewRequest: ", err.Error())
				}
				req.Header = r.Header
				log.Println(httpClient.Do(req))

			}
		}
	})

	fmt.Println("[INFO] Server listening")
	http.ListenAndServe(":5000", nil)
}
