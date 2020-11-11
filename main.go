package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// Discord color values
const (
	ColorRed   = 10038562
	ColorGreen = 3066993
	ColorGrey  = 9807270
)

type alertManAlert struct {
	Annotations struct {
		Description string `json:"description"`
		Summary     string `json:"summary"`
	} `json:"annotations"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Labels       map[string]string `json:"labels"`
	StartsAt     string            `json:"startsAt"`
	Status       string            `json:"status"`
}

type alertManOut struct {
	Alerts            []alertManAlert `json:"alerts"`
	CommonAnnotations struct {
		Summary string `json:"summary"`
		Description string `json:"description"`
	} `json:"commonAnnotations"`
	CommonLabels struct {
		Alertname string `json:"alertname"`
	} `json:"commonLabels"`
	ExternalURL string `json:"externalURL"`
	GroupKey    string `json:"groupKey"`
	GroupLabels struct {
		Alertname string `json:"alertname"`
	} `json:"groupLabels"`
	Receiver string `json:"receiver"`
	Status   string `json:"status"`
	Version  string `json:"version"`
}

type discordOut struct {
	Content string         `json:"content"`
	Embeds  []discordEmbed `json:"embeds"`
}

type discordEmbed struct {
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Color       int                 `json:"color"`
	Fields      []discordEmbedField `json:"fields"`
}

type discordEmbedField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func main() {
	webhookUrl := os.Getenv("DISCORD_WEBHOOK")
	whURL := flag.String("webhook.url", webhookUrl, "")
	discordRole := os.Getenv("DISCORD_ROLE")
	dcRole := flag.String("webhook.role", discordRole, "")
	flag.Parse()

	if webhookUrl == "" && *whURL == "" {
		fmt.Fprintf(os.Stderr, "error: environment variable DISCORD_WEBHOOK not found\n")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "info: Listening on 0.0.0.0:9094\n")
	http.ListenAndServe(":9094", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			panic(err)
		}

		amo := alertManOut{}
		err = json.Unmarshal(b, &amo)
		if err != nil {
			panic(err)
		}

		groupedAlerts := make(map[string][]alertManAlert)

		for _, alert := range amo.Alerts {
			groupedAlerts[alert.Status] = append(groupedAlerts[alert.Status], alert)
		}

		for status, alerts := range groupedAlerts {
			DO := discordOut{}

			RichEmbed := discordEmbed{
				Title:       fmt.Sprintf("[%s:%d] %s", strings.ToUpper(status), len(alerts), amo.CommonLabels.Alertname),
				Description: amo.CommonAnnotations.Description,
				Color:       ColorGrey,
				Fields:      []discordEmbedField{},
			}

			if status == "firing" {
				RichEmbed.Color = ColorRed
			} else if status == "resolved" {
				RichEmbed.Color = ColorGreen
			}
			
			if amo.CommonAnnotations.Summary != "" {
				if *dcRole == "" {
					DO.Content = fmt.Sprintf(" === %s === \n", amo.CommonAnnotations.Summary)
				} else {
					DO.Content = fmt.Sprintf("%s\n", *dcRole)
				}
			}
			
			DO.Embeds = []discordEmbed{RichEmbed}

			DOD, _ := json.Marshal(DO)
			http.Post(*whURL, "application/json", bytes.NewReader(DOD))
		}
	}))
}
