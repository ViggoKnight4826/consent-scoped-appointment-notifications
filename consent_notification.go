package consentnotification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const canonicalImport = "infrai.auth.consent.check"

type Appointment struct {
	UserID   string
	Category string
	Message  string
}

type Decision struct {
	Send   bool
	Reason string
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type consentData struct {
	Granted bool `json:"granted"`
}

type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{BaseURL: baseURL, APIKey: key, HTTPClient: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *Client) CheckConsent(ctx context.Context, userID, category string) (bool, error) {
	path := fmt.Sprintf("/v1/auth/consent/check/%s/%s", userID, category)
	var data consentData
	if err := c.do(ctx, http.MethodGet, path, nil, &data); err != nil {
		return false, err
	}
	return data.Granted, nil
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, data any) error {
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Accept", "application/json")
		res, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		var env envelope
		decodeErr := json.NewDecoder(res.Body).Decode(&env)
		res.Body.Close()
		if decodeErr != nil {
			return decodeErr
		}
		if !env.OK {
			if env.Error == nil {
				return fmt.Errorf("infrai request rejected with HTTP %d", res.StatusCode)
			}
			return fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			delay := time.Duration(1<<attempt) * 100 * time.Millisecond
			if retryAfter := res.Header.Get("Retry-After"); retryAfter != "" {
				if seconds, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil {
					delay = seconds
				}
			}
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai transport returned HTTP %d", res.StatusCode)
		}
		if data != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, data)
		}
		return nil
	}
	return errors.New("request retry limit reached")
}

func Decide(ctx context.Context, c *Client, a Appointment) (Decision, error) {
	granted, err := c.CheckConsent(ctx, a.UserID, a.Category)
	if err != nil {
		return Decision{}, err
	}
	if !granted {
		return Decision{Reason: "consent is not granted"}, nil
	}
	return Decision{Send: true, Reason: "consent is granted"}, nil
}
