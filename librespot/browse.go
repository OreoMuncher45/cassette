package librespot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (l *LibrespotApiClient) Browse(ctx context.Context, kind string, query url.Values, result any) error {
	path := "/browse/" + url.PathEscape(kind)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	resp, err := l.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return fmt.Errorf("Spotify Connect is not paired yet")
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("browse %s: daemon returned HTTP %d", kind, resp.StatusCode)
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return fmt.Errorf("decode browse %s: %w", kind, err)
	}
	field := "items"
	if kind == "user" {
		field = "id"
	}
	if kind == "first-track" {
		field = "uri"
	}
	if _, ok := raw[field]; !ok {
		return fmt.Errorf("browse %s: incompatible daemon response; rebuild the patched daemon", kind)
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}
