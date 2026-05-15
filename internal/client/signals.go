package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/junghoonkye/tossinvest-cli/internal/domain"
)

type signalsBatchEnvelope struct {
	Result struct {
		Signals []struct {
			ProductCode          string `json:"productCode"`
			ReasoningDescription string `json:"reasoningDescription"`
		} `json:"signals"`
	} `json:"result"`
}

// ListSignals returns one-line Korean reasoning per code in the input list.
// Toss documents no explicit cap; observed batches of 100 codes return 200 ok.
func (c *Client) ListSignals(ctx context.Context, productCodes []string) ([]domain.Signal, error) {
	cleaned := make([]string, 0, len(productCodes))
	for _, code := range productCodes {
		code = strings.TrimSpace(code)
		if code != "" {
			cleaned = append(cleaned, code)
		}
	}
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("ListSignals: at least one product code is required")
	}

	body, err := json.Marshal(map[string]any{"productCodes": cleaned})
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/api/v1/dashboard/wts/overview/ai-signals", c.infoBaseURL)
	var envelope signalsBatchEnvelope
	if err := c.postJSON(ctx, endpoint, body, &envelope); err != nil {
		return nil, err
	}

	out := make([]domain.Signal, 0, len(envelope.Result.Signals))
	for _, s := range envelope.Result.Signals {
		out = append(out, domain.Signal{
			ProductCode:          s.ProductCode,
			ReasoningDescription: s.ReasoningDescription,
		})
	}
	return out, nil
}

// Suppress unused-import warnings until later tasks fill in.
var (
	_ = url.Parse
	_ = time.Now
)
