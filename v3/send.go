package sn

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mailjet/mailjet-apiv3-go"
)

func getMJKeys() (string, string) {
	return os.Getenv("MJ_API_KEY_PUB"), os.Getenv("MJ_API_KEY_PRIV")
}

// SendMessages sends email messages
func SendMessages(msgInfo ...mailjet.InfoMessagesV31) (*mailjet.ResultsV31, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	pub, priv := getMJKeys()
	mailjetClient := mailjet.NewMailjetClient(pub, priv)
	msgs := mailjet.MessagesV31{Info: msgInfo}
	if IsProduction() {
		return mailjetClient.SendMailV31(&msgs)
	}
	for _, msg := range msgInfo {
		slog.Debug(fmt.Sprintf("sent message: %#v", msg))
	}
	return nil, nil
}
