package main

import (
	trans "github.com/boryashkin/instractor/translation"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

var translations = trans.TranslationMap{}

const (
	EN = "en"
	RU = "ru"
)
const (
	ERR_DOESNT_SUPPORT_COMMANDS    = 1
	EN_ERR_DOESNT_SUPPORT_COMMANDS = "The bot doesn't support commands yet. Only links to instagram posts."
	RU_ERR_DOESNT_SUPPORT_COMMANDS = "Бот пока не поддерживает комманды. Только ссылки на посты из инстаграма."
	ERR_INVALID_URL                = 2
	EN_ERR_INVALID_URL             = "Invalid url, try again."
	RU_ERR_INVALID_URL             = "Невалидная ссылка, попробуйте ещё раз."
	ERR_NOT_INSTAGRAM_URL          = 3
	EN_ERR_NOT_INSTAGRAM_URL       = "It's not an instagram url, try again."
	RU_ERR_NOT_INSTAGRAM_URL       = "Это ссылка не на инстаграм, попробуйте ещё раз."
	ERR_FAILED_TO_GET_RESPONSE     = 4
	EN_ERR_FAILED_TO_GET_RESPONSE  = "Failed to get a response from the link."
	RU_ERR_FAILED_TO_GET_RESPONSE  = "Не удалось получить ответ по ссылке."
	ERR_FAILED_TO_READ_RESPONSE    = 5
	EN_ERR_FAILED_TO_READ_RESPONSE = "Failed to read a response from the link."
	RU_ERR_FAILED_TO_READ_RESPONSE = "Не удалось прочитать ответ по ссылке."
)

func main() {
	tgtoken := os.Getenv("TGTOKEN")

	bot, err := tgbotapi.NewBotAPI(tgtoken)
	if err != nil {
		log.Panic(err)
	}
	initTranslations(translations)
	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}
		lang := update.Message.From.LanguageCode
		if lang != EN && lang != RU {
			lang = EN
		}
		go handleAsync(update, bot, lang)
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

func handleAsync(update tgbotapi.Update, bot *tgbotapi.BotAPI, lang string) {
	log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

	if update.Message.IsCommand() {
		update.Message.Text = translations[lang][ERR_DOESNT_SUPPORT_COMMANDS]
	} else {
		handleInstaUrl(&update, lang)
	}

	msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
	msg.ReplyToMessageID = update.Message.MessageID
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send a response [%s]", err.Error())
	}
}

func handleInstaUrl(update *tgbotapi.Update, lang string) *tgbotapi.Update {
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
		update.Message.Text = translations[lang][ERR_INVALID_URL]
	} else if !validateInstaUrl(recvUrl) {
		update.Message.Text = translations[lang][ERR_NOT_INSTAGRAM_URL]
	} else {
		update.Message.Text = extractInstaTextFromUrl(recvUrl, lang)
	}

	return update
}

func extractInstaTextFromUrl(recvUrl *url.URL, lang string) string {
	recvUrl = addJsonRequestParam(recvUrl)
	log.Printf("Requesting [%s]", recvUrl.String())
	resp, err := http.Get(recvUrl.String())
	if err != nil || resp == nil {
		log.Println("error", err, "resoponse", resp)
		return translations[lang][ERR_FAILED_TO_GET_RESPONSE]
	}

	text := ""
	log.Printf("Reading body")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		text = translations[lang][ERR_FAILED_TO_READ_RESPONSE]
	} else {
		text = generateResponseFromJsonBody(string(body), lang)
	}

	err = resp.Body.Close()
	if err != nil {
		log.Println("Failed to close Body")
	}

	return text
}

func generateResponseFromJsonBody(body string, lang string) string {
	log.Printf("Extracting json xpath")
	descriptionText := gjson.Get(string(body), "graphql.shortcode_media.edge_media_to_caption.edges.0.node.text")
	log.Printf("Extracted result [%s]", descriptionText.String())
	if !descriptionText.Exists() {
		return translations[lang][ERR_FAILED_TO_READ_RESPONSE]
	}

	return descriptionText.String()
}

func initTranslations(translations trans.TranslationMap) {
	translations.InitLangMap(RU)
	translations.InitLangMap(EN)
	translations.AddTranslation(RU, ERR_DOESNT_SUPPORT_COMMANDS, RU_ERR_DOESNT_SUPPORT_COMMANDS)
	translations.AddTranslation(EN, ERR_DOESNT_SUPPORT_COMMANDS, EN_ERR_DOESNT_SUPPORT_COMMANDS)
	translations.AddTranslation(RU, ERR_INVALID_URL, RU_ERR_INVALID_URL)
	translations.AddTranslation(EN, ERR_INVALID_URL, EN_ERR_INVALID_URL)
	translations.AddTranslation(RU, ERR_NOT_INSTAGRAM_URL, RU_ERR_NOT_INSTAGRAM_URL)
	translations.AddTranslation(EN, ERR_NOT_INSTAGRAM_URL, EN_ERR_NOT_INSTAGRAM_URL)
	translations.AddTranslation(RU, ERR_FAILED_TO_GET_RESPONSE, RU_ERR_FAILED_TO_GET_RESPONSE)
	translations.AddTranslation(EN, ERR_FAILED_TO_GET_RESPONSE, EN_ERR_FAILED_TO_GET_RESPONSE)
	translations.AddTranslation(RU, ERR_FAILED_TO_READ_RESPONSE, RU_ERR_FAILED_TO_READ_RESPONSE)
	translations.AddTranslation(EN, ERR_FAILED_TO_READ_RESPONSE, EN_ERR_FAILED_TO_READ_RESPONSE)
	log.Println("translations are initialized")
}
