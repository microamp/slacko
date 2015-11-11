package utils

import (
	"encoding/json"
	"fmt"
	"html"
	"log"
	"regexp"
	"strings"

	"github.com/microamp/slacko/config"
	"github.com/nlopes/slack"
)

const msgSubTypeMessageChanged = "message_changed"

type MessageContext struct {
	*slack.Client
	MsgChannel   string
	MsgTimestamp string
	MsgUser      string
	MsgText      string
	MsgEdited    bool
	ReplyToID    string
	Host         string
	BotName      string
	DebugOn      bool
}

func NewMessageContext(client *slack.Client, event *slack.MessageEvent, config *config.SlackoConfig) *MessageContext {
	edited := event.Msg.SubType == msgSubTypeMessageChanged
	var msgText, msgTimestamp string
	if edited {
		msgText, msgTimestamp = event.SubMessage.Text, event.SubMessage.Timestamp
	} else {
		msgText, msgTimestamp = event.Msg.Text, event.Msg.Timestamp
	}

	return &MessageContext{
		Client:       client,
		MsgChannel:   event.Msg.Channel,
		MsgTimestamp: msgTimestamp,
		MsgUser:      event.Msg.User,
		MsgText:      msgText,
		MsgEdited:    edited,
		Host:         config.GoPlaygroundHost,
		BotName:      config.BotName,
		DebugOn:      config.DebugOn,
	}
}

func (mc *MessageContext) Printf(format string, v ...interface{}) {
	if mc.DebugOn {
		log.Printf(format, v...)
	}
}

func (mc *MessageContext) GetInfo() (string, error) {
	jsonified, err := json.Marshal(mc)
	if err != nil {
		return "", err
	}
	return string(jsonified), nil
}

func (mc *MessageContext) IsBot() (bool, error) {
	userInfo, err := mc.GetUserInfo(mc.MsgUser)
	if err != nil {
		return false, err
	}
	return userInfo.IsBot, nil
}

func (mc *MessageContext) ExtractReplyToID() string {
	pattern := regexp.MustCompile(`^<@([^/]+)>:`)
	groups := pattern.FindStringSubmatch(mc.MsgText)
	if groups == nil {
		return ""
	}
	return groups[1]
}

func (mc *MessageContext) ExtractCode(botID string) string {
	prefix := fmt.Sprintf("<@%s>:", botID)
	temp := strings.TrimSpace(strings.Trim(mc.MsgText, prefix))
	if !strings.HasPrefix(temp, "`") || !strings.HasSuffix(temp, "`") {
		return ""
	}
	temp = strings.Trim(temp, "`")
	return html.UnescapeString(temp) // e.g. "&lt;=" to "<="
}
