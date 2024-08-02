package sn

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"firebase.google.com/go/v4/messaging"
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

func (cl *GameClient[GT, G]) sendNotifications(ctx context.Context, g G, pids []PID) (*messaging.BatchResponse, error) {
	slog.Debug(msgEnter)
	defer slog.Debug(msgExit)

	tokens, err := cl.getTokenStrings(ctx, g.getHeader().ID, g.UIDSForPIDS(pids))
	if err != nil {
		return nil, err
	}
	slog.Debug(fmt.Sprintf("tokens: %v", tokens))
	if len(tokens) < 1 {
		return nil, nil
	}
	notifications := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title:    "It is your turn at SlothNinja Games",
			Body:     "One or more games await your move.",
			ImageURL: "https://www.slothninja.com/public/logo.png",
		},
		Webpush: &messaging.WebpushConfig{
			FCMOptions: &messaging.WebpushFCMOptions{
				Link: "https://www.slothninja.com",
			},
		},
	}
	return cl.FCM.SendEachForMulticast(ctx, notifications)
}
