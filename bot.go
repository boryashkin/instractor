package main

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {
	tgtoken := os.Getenv("TGTOKEN")

	bot, err := tgbotapi.NewBotAPI(tgtoken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}

		go handleAsync(update, bot)
	}

	log.Printf("After the loop")
}

func validateInstaUrl(recvUrl *url.URL) bool {
	return recvUrl.Host == "instagram.com" || recvUrl.Host == "www.instagram.com"
}

func addJsonRequestParam(recvUrl *url.URL) *url.URL {
	values := recvUrl.Query()

	values.Set("__a", "1")

	recvUrl.RawQuery = values.Encode()

	return recvUrl
}

func handleAsync(update tgbotapi.Update, bot *tgbotapi.BotAPI) {
	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
	if update.Message.IsCommand() {
		update.Message.Text = "The bot doesn't support commands yet. Only links to instagram posts."
	} else {
		handleInstaUrl(&update)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
	msg.ReplyToMessageID = update.Message.MessageID
	bot.Send(msg)
}

func handleInstaUrl(update *tgbotapi.Update) *tgbotapi.Update {
	log.Printf("Got [%s], parsing", update.Message.Text)

	recvUrl, err := url.Parse(update.Message.Text)
	if err != nil || recvUrl == nil {
		log.Println("first cond")
		update.Message.Text = "Invalid url, try again."
	} else if !validateInstaUrl(recvUrl) {
		log.Println("sec cond")
		update.Message.Text = "It's not an instagram url, try again."
	} else {
		log.Println("else")
		recvUrl = addJsonRequestParam(recvUrl)
		log.Printf("Requesting [%s]", recvUrl.String())
		resp, err := http.Get(recvUrl.String())
		if err != nil || resp == nil {
			log.Println("error", err, "resoponse", resp)
			update.Message.Text = "Failed to get a response from the link."
		} else {
			log.Printf("Reading body")
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				update.Message.Text = "Failed to read a response from the link."
			} else {
				log.Printf("Extracting json xpath")
				descriptionText := gjson.Get(string(body), "graphql.shortcode_media.edge_media_to_caption.edges.0.node.text")
				log.Printf("Extracted result", descriptionText.Raw, descriptionText.String())
				if !descriptionText.Exists() {
					update.Message.Text = "Failed to read a response from the link."
				} else {
					update.Message.Text = descriptionText.String()
				}
			}

			resp.Body.Close()
		}
	}

	return update
}
