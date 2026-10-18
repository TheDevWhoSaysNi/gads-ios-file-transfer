package transfer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProviderLaunch asks a GADS provider to open the app.
// The provider already owns the iOS 17 tunnel, so this does not open a second one.
func ProviderLaunch(baseURL string) LaunchFunc {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	client := &http.Client{Timeout: 30 * time.Second}
	return func(ctx context.Context, udid string) error {
		launchURL := base + "/device/" + url.PathEscape(udid) + "/launchApp"
		body := bytes.NewBufferString(`{"app":"` + BundleID + `"}`)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, launchURL, body)
		if err != nil {
			return err
		}
		req.Header.Set("content-type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("provider launch: %w", err)
		}
		defer res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			msg, _ := io.ReadAll(io.LimitReader(res.Body, 400))
			return fmt.Errorf("provider launch returned %d: %s", res.StatusCode, strings.TrimSpace(string(msg)))
		}
		return nil
	}
}

// ProviderHome asks the GADS provider to leave the app and return to the home screen.
func ProviderHome(baseURL, udid string) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" || udid == "" {
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	homeURL := base + "/device/" + url.PathEscape(udid) + "/home"
	res, err := client.Post(homeURL, "application/json", nil)
	if err == nil {
		res.Body.Close()
	}
}
