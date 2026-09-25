package max

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/abelith/etu-bot/internal/models"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	"net/http"
)

const baseURL = "https://platform-api2.max.ru"

type Notifier struct {
	BotToken string
}

func (n *Notifier) Notify(ctx context.Context, task models.NotificationTask) error {
	if _, err := forwardMessage(ctx, n.BotToken, task.UserID, task.SourceID); err != nil {
		return err
	}
	return nil
}

func forwardMessage(ctx context.Context, token string, userID int, sourceID string) (model.SendMessageResult, error) {
	body := model.NewMessageBody{
		Link: &model.NewMessageLink{
			Type: model.LinkTypeForward,
			Mid:  sourceID,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return model.SendMessageResult{}, fmt.Errorf("marshal body: %w", err)
	}

	url := fmt.Sprintf("%s/messages?user_id=%d", baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return model.SendMessageResult{}, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return model.SendMessageResult{}, fmt.Errorf("failed to do request: %w", err)
	}
	defer resp.Body.Close()

	var res model.SendMessageResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return model.SendMessageResult{}, fmt.Errorf("failed to decode response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return res, fmt.Errorf("max api error, status %d", resp.StatusCode)
	}
	return res, nil
}
