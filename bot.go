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
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send a response [%s]", err.Error())
	}
}

func handleInstaUrl(update *tgbotapi.Update) *tgbotapi.Update {
	log.Printf(
		"Got [%s], parsing, uid [%d], fname [%s], username [%s], lang [%s]",
		update.Message.Text,
		update.Message.From.ID,
		update.Message.From.FirstName,
		update.Message.From.UserName,
		update.Message.From.LanguageCode,
	)

	recvUrl, err := url.Parse(update.Message.Text)
	if err != nil || recvUrl == nil {
		update.Message.Text = "Invalid url, try again."
	} else if !validateInstaUrl(recvUrl) {
		update.Message.Text = "It's not an instagram url, try again."
	} else {
		update.Message.Text = extractInstaTextFromUrl(recvUrl)
	}

	return update
}

func extractInstaTextFromUrl(recvUrl *url.URL) string {
	recvUrl = addJsonRequestParam(recvUrl)
	log.Printf("Requesting [%s]", recvUrl.String())
	resp, err := http.Get(recvUrl.String())
	if err != nil || resp == nil {
		log.Println("error", err, "resoponse", resp)
		return "Failed to get a response from the link."
	}

	text := ""
	log.Printf("Reading body")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		text = "Failed to read a response from the link."
	} else {
		text = generateResponseFromJsonBody(string(body))
	}

	err = resp.Body.Close()
	if err != nil {
		log.Println("Failed to close Body")
	}

	return text
}

func generateResponseFromJsonBody(body string) string {
	log.Printf("Extracting json xpath")
	descriptionText := gjson.Get(string(body), "graphql.shortcode_media.edge_media_to_caption.edges.0.node.text")
	log.Printf("Extracted result [%s]", descriptionText.String())
	if !descriptionText.Exists() {
		return "Failed to read a response from the link."
	}

	return descriptionText.String()
}
