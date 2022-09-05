package sn

import (
	"context"
	"os"

	"github.com/SlothNinja/log"
	"github.com/mailjet/mailjet-apiv3-go"
)

func getMJKeys() (string, string) {
	return os.Getenv("MJ_API_KEY_PUB"), os.Getenv("MJ_API_KEY_PRIV")
}

func SendMessages(c context.Context, msgInfo ...mailjet.InfoMessagesV31) (*mailjet.ResultsV31, error) {
	log.Debugf("Entering")
	defer log.Debugf("Entering")

	pub, priv := getMJKeys()
	mailjetClient := mailjet.NewMailjetClient(pub, priv)
	msgs := mailjet.MessagesV31{Info: msgInfo}
	if IsProduction() {
		return mailjetClient.SendMailV31(&msgs)
	}
	for _, msg := range msgInfo {
		log.Debugf("sent message: %#v", msg)
	}
	return nil, nil
}
