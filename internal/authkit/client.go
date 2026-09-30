package authkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/panoratech/twinbay-cli/internal/clierrors"
)

const (
	ClientID       = "client_01KZFJ6RGBBJWJ8Z7QSZ2YA57Y"
	UserManagement = "https://api.workos.com/user_management"
	deviceGrant    = "urn:ietf:params:oauth:grant-type:device_code"
	requestTimeout = 30 * time.Second
)

type DeviceAuthorization struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type TokenResponse struct {
	User           User   `json:"user"`
	OrganizationID string `json:"organization_id"`
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
}

type OAuthError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
	StatusCode  int    `json:"-"`
}

func (e *OAuthError) Error() string {
	if e.Description != "" {
		return e.Description
	}
	if e.Code != "" {
		return strings.ReplaceAll(e.Code, "_", " ")
	}
	return fmt.Sprintf("authentication request failed (%d)", e.StatusCode)
}

type Client struct {
	ClientID   string
	BaseURL    string
	HTTPClient *http.Client
	Sleep      func(context.Context, time.Duration) error
}

func NewClient() *Client {
	return &Client{
		ClientID:   ClientID,
		BaseURL:    UserManagement,
		HTTPClient: &http.Client{Timeout: requestTimeout},
		Sleep:      sleepContext,
	}
}

func (c *Client) AuthorizeDevice(ctx context.Context) (*DeviceAuthorization, error) {
	var out DeviceAuthorization
	if err := c.postForm(ctx, "/authorize/device", url.Values{"client_id": {c.ClientID}}, &out); err != nil {
		return nil, fmt.Errorf("start browser login: %w", err)
	}
	if out.DeviceCode == "" || out.UserCode == "" || out.VerificationURI == "" {
		return nil, errors.New("start browser login: WorkOS returned an incomplete device authorization")
	}
	if out.ExpiresIn <= 0 {
		out.ExpiresIn = 300
	}
	if out.Interval <= 0 {
		out.Interval = 5
	}
	return &out, nil
}

func (c *Client) Poll(ctx context.Context, device *DeviceAuthorization) (*TokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(device.ExpiresIn)*time.Second)
	defer cancel()

	interval := time.Duration(device.Interval) * time.Second
	for {
		var out TokenResponse
		err := c.postForm(ctx, "/authenticate", url.Values{
			"grant_type":  {deviceGrant},
			"device_code": {device.DeviceCode},
			"client_id":   {c.ClientID},
		}, &out)
		if err == nil {
			if out.AccessToken == "" || out.RefreshToken == "" {
				return nil, errors.New("browser login: WorkOS returned an incomplete session")
			}
			return &out, nil
		}

		var oauthErr *OAuthError
		if !errors.As(err, &oauthErr) {
			return nil, fmt.Errorf("browser login: %w", err)
		}
		switch oauthErr.Code {
		case "authorization_pending":
		case "slow_down":
			interval += time.Second
		case "access_denied":
			return nil, authenticationError(errors.New("browser login was denied"))
		case "expired_token":
			return nil, authenticationError(errors.New("browser login expired; run twinbay auth login to try again"))
		default:
			return nil, authenticationError(fmt.Errorf("browser login: %w", oauthErr))
		}
		if err := c.Sleep(ctx, interval); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil, authenticationError(errors.New("browser login expired; run twinbay auth login to try again"))
			}
			return nil, fmt.Errorf("browser login canceled: %w", err)
		}
	}
}

func authenticationError(err error) error {
	return clierrors.WithExitCode(err, clierrors.ExitAuth)
}

func (c *Client) Refresh(ctx context.Context, refreshToken, organizationID string) (*TokenResponse, error) {
	values := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.ClientID},
	}
	if organizationID != "" {
		values.Set("organization_id", organizationID)
	}
	var out TokenResponse
	if err := c.postForm(ctx, "/authenticate", values, &out); err != nil {
		return nil, fmt.Errorf("refresh browser session: %w", err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		return nil, errors.New("refresh browser session: WorkOS returned an incomplete session")
	}
	return &out, nil
}

func (c *Client) Logout(ctx context.Context, accessToken string) error {
	claims, err := ParseTokenClaims(accessToken)
	if err != nil || claims.SessionID == "" {
		return nil
	}
	u := strings.TrimRight(c.BaseURL, "/") + "/sessions/logout?" + url.Values{"session_id": {claims.SessionID}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	httpClient := *c.httpClient()
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	res, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("end browser session: %w", err)
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusFound {
		return fmt.Errorf("end browser session: WorkOS returned %s", res.Status)
	}
	return nil
}

func (c *Client) postForm(ctx context.Context, path string, values url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+path, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		oauthErr := &OAuthError{StatusCode: res.StatusCode}
		_ = json.Unmarshal(body, oauthErr)
		return oauthErr
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode WorkOS response: %w", err)
	}
	return nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// BrowserCommand returns the platform browser launcher and arguments.
func BrowserCommand(target string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{target}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", target}
	default:
		return "xdg-open", []string{target}
	}
}
