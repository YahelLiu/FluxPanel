package messaging

import (
	"context"
	"fmt"
	"log"

	"client-monitor/ilink"

	"github.com/google/uuid"
)

// NewClientID generates a new unique client ID for message correlation.
func NewClientID() string {
	return uuid.New().String()
}

// SendTypingState sends a typing indicator to a user via the iLink sendtyping API.
// It first fetches a typing_ticket via getconfig, then sends the typing status.
func SendTypingState(ctx context.Context, client *ilink.Client, userID, contextToken string) error {
	// Get typing ticket
	configResp, err := client.GetConfig(ctx, userID, contextToken)
	if err != nil {
		return fmt.Errorf("get config for typing: %w", err)
	}
	if configResp.TypingTicket == "" {
		return fmt.Errorf("no typing_ticket returned from getconfig")
	}

	// Send typing
	if err := client.SendTyping(ctx, userID, configResp.TypingTicket, ilink.TypingStatusTyping); err != nil {
		return fmt.Errorf("send typing: %w", err)
	}

	log.Printf("[sender] sent typing indicator to %s", userID)
	return nil
}

// SendTextReply sends a text reply to a user through the iLink API.
// If clientID is empty, a new one is generated.
// If contextToken is empty, it will try to get one via getconfig.
func SendTextReply(ctx context.Context, client *ilink.Client, toUserID, text, contextToken, clientID string) error {
	if clientID == "" {
		clientID = NewClientID()
	}

	// Convert markdown to plain text for WeChat display
	plainText := MarkdownToPlainText(text)

	botID := client.BotID()
	log.Printf("[sender] sending message: from=%s, to=%s, hasContextToken=%v", botID, toUserID, contextToken != "")

	// If no context token, try to get one from getconfig
	if contextToken == "" {
		log.Printf("[sender] no context token, fetching from getconfig for %s", toUserID)
		configResp, err := client.GetConfig(ctx, toUserID, "")
		if err != nil {
			log.Printf("[sender] getconfig failed: %v, trying to send anyway", err)
		} else {
			log.Printf("[sender] getconfig response: ret=%d, hasTypingTicket=%v", configResp.Ret, configResp.TypingTicket != "")
		}
	}

	req := &ilink.SendMessageRequest{
		Msg: ilink.SendMsg{
			FromUserID:   botID,
			ToUserID:     toUserID,
			ClientID:     clientID,
			MessageType:  ilink.MessageTypeBot,
			MessageState: ilink.MessageStateFinish,
			ItemList: []ilink.MessageItem{
				{
					Type: ilink.ItemTypeText,
					TextItem: &ilink.TextItem{
						Text: plainText,
					},
				},
			},
			ContextToken: contextToken,
		},
		BaseInfo: ilink.BaseInfo{},
	}

	resp, err := client.SendMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	if resp.Ret != 0 {
		return fmt.Errorf("send message failed: ret=%d errmsg=%s", resp.Ret, resp.ErrMsg)
	}

	log.Printf("[sender] sent reply to %s: success", toUserID)
	return nil
}
