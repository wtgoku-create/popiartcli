package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wtgoku-create/popiartcli/internal/output"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

type RequestOptions struct {
	Query  map[string]string
	Body   any
	Token  string
	Accept string
}

type envelope struct {
	OK    *bool           `json:"ok,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error json.RawMessage `json:"error,omitempty"`
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) GetJSON(ctx context.Context, path string, query map[string]string, dst any) error {
	return c.doJSON(ctx, http.MethodGet, path, RequestOptions{Query: query}, dst)
}

func (c *Client) PostJSON(ctx context.Context, path string, body any, dst any) error {
	return c.doJSON(ctx, http.MethodPost, path, RequestOptions{Body: body}, dst)
}

func (c *Client) DeleteJSON(ctx context.Context, path string, body any, dst any) error {
	return c.doJSON(ctx, http.MethodDelete, path, RequestOptions{Body: body}, dst)
}

func (c *Client) Stream(ctx context.Context, method, path string, opts RequestOptions) (*http.Response, error) {
	req, err := c.newRequest(ctx, method, path, opts)
	if err != nil {
		return nil, err
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, output.NewError("NETWORK_ERROR", fmt.Sprintf("Request failed: %v", err), map[string]any{
			"url": req.URL.String(),
		})
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		defer res.Body.Close()
		return nil, c.decodeError(res)
	}

	return res, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, opts RequestOptions, dst any) error {
	req, err := c.newRequest(ctx, method, path, opts)
	if err != nil {
		return err
	}

	res, err := c.HTTP.Do(req)
	if err != nil {
		return output.NewError("NETWORK_ERROR", fmt.Sprintf("Request failed: %v", err), map[string]any{
			"url": req.URL.String(),
		})
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return c.decodeError(res)
	}

	if dst == nil {
		io.Copy(io.Discard, res.Body)
		return nil
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return output.NewError("NETWORK_ERROR", "Failed to read response body", nil)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err == nil && env.OK != nil {
		if *env.OK && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, dst)
		}
		if !*env.OK && len(env.Error) > 0 {
			var details map[string]any
			if json.Unmarshal(env.Error, &details) == nil {
				code := asString(details["code"], httpStatusCode(res.StatusCode))
				message := asString(details["message"], fmt.Sprintf("HTTP %d", res.StatusCode))
				delete(details, "code")
				delete(details, "message")
				details["status"] = res.StatusCode
				return output.NewError(code, message, details)
			}
		}
	}

	return json.Unmarshal(body, dst)
}

func (c *Client) newRequest(ctx context.Context, method, path string, opts RequestOptions) (*http.Request, error) {
	base, err := url.Parse(c.BaseURL + "/")
	if err != nil {
		return nil, output.NewError("BAD_REQUEST", "Invalid API endpoint", map[string]any{
			"endpoint": c.BaseURL,
		})
	}

	ref, err := url.Parse(strings.TrimLeft(path, "/"))
	if err != nil {
		return nil, output.NewError("BAD_REQUEST", "Invalid API path", map[string]any{
			"path": path,
		})
	}

	u := base.ResolveReference(ref)
	if len(opts.Query) > 0 {
		q := u.Query()
		for key, value := range opts.Query {
			if value == "" {
				continue
			}
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
	}

	var bodyReader io.Reader
	if opts.Body != nil {
		payload, err := json.Marshal(opts.Body)
		if err != nil {
			return nil, output.NewError("INPUT_PARSE_ERROR", "Failed to encode request body", nil)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, output.NewError("BAD_REQUEST", "Failed to build request", nil)
	}

	req.Header.Set("Accept", defaultString(opts.Accept, "application/json"))
	req.Header.Set("User-Agent", "popiart-cli/0.1.0")
	if opts.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	token := opts.Token
	if token == "" {
		token = c.Token
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return req, nil
}

func (c *Client) decodeError(res *http.Response) error {
	body, _ := io.ReadAll(res.Body)
	body = bytes.TrimSpace(body)

	if len(body) == 0 {
		return output.NewError(httpStatusCode(res.StatusCode), fmt.Sprintf("HTTP %d", res.StatusCode), map[string]any{
			"status": res.StatusCode,
		})
	}

	var env envelope
	if err := json.Unmarshal(body, &env); err == nil && env.OK != nil && !*env.OK && len(env.Error) > 0 {
		var details map[string]any
		if json.Unmarshal(env.Error, &details) == nil {
			code := asString(details["code"], httpStatusCode(res.StatusCode))
			message := asString(details["message"], fmt.Sprintf("HTTP %d", res.StatusCode))
			delete(details, "code")
			delete(details, "message")
			details["status"] = res.StatusCode
			return output.NewError(code, message, details)
		}
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err == nil {
		code := httpStatusCode(res.StatusCode)
		message := asString(payload["message"], fmt.Sprintf("HTTP %d", res.StatusCode))
		payload["status"] = res.StatusCode
		return output.NewError(code, message, payload)
	}

	return output.NewError(httpStatusCode(res.StatusCode), string(body), map[string]any{
		"status": res.StatusCode,
	})
}

func httpStatusCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "BAD_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusUnprocessableEntity:
		return "VALIDATION_ERROR"
	case http.StatusTooManyRequests:
		return "RATE_LIMITED"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	case http.StatusInternalServerError:
		return "SERVER_ERROR"
	default:
		return "HTTP_ERROR"
	}
}

func asString(value any, fallback string) string {
	if s, ok := value.(string); ok && s != "" {
		return s
	}
	return fallback
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
