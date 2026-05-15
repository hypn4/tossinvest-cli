package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// postJSONEmpty issues a POST request with body `{}` and decodes the JSON
// response into dst. Used by Toss endpoints that gate on a non-empty request
// body (e.g. /api/v2/stock-infos/evaluation/{code}).
func (c *Client) postJSONEmpty(ctx context.Context, endpoint string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", DefaultBrowserUserAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("POST %s: %s — %s", endpoint, resp.Status, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}
