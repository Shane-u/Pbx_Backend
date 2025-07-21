package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"sync"
)

type SiliconFlowHandler struct {
	mutex    sync.Mutex
	APIKey   string
	Endpoint string
	Model    string
	ctx      context.Context
	logger   *logrus.Logger
}

type SFMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type SFRequest struct {
	Model     string      `json:"model"`
	Messages  []SFMessage `json:"messages"`
	MaxTokens int         `json:"max_tokens"`
}

type SFChoice struct {
	Message SFMessage `json:"message"`
}

type SFResponse struct {
	Choices []SFChoice `json:"choices"`
}

func NewSiliconFlowHandler(ctx context.Context, apiKey, endpoint, model string, logger *logrus.Logger) *SiliconFlowHandler {
	return &SiliconFlowHandler{
		ctx:      ctx,
		APIKey:   apiKey,
		Endpoint: endpoint,
		Model:    model,
		logger:   logger,
	}
}

func (h *SiliconFlowHandler) Query(userMsg string) (string, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	reqBody := SFRequest{
		Model:     h.Model,
		Messages:  []SFMessage{{Role: "user", Content: userMsg}},
		MaxTokens: 128,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequestWithContext(h.ctx, "POST", h.Endpoint, bytes.NewReader(body))
	req.Header.Set("accept", "application/json")
	req.Header.Set("authorization", "Bearer "+h.APIKey)
	req.Header.Set("content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var sfResp SFResponse
	if err := json.Unmarshal(respBody, &sfResp); err != nil {
		return "", err
	}
	if len(sfResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return sfResp.Choices[0].Message.Content, nil
}
