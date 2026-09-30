package authkit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/panoratech/twinbay-cli/internal/clierrors"
	"github.com/panoratech/twinbay-cli/internal/config"
	"github.com/zalando/go-keyring"
)

const sessionKey = "authkit-session"

var ErrNoSession = clierrors.WithExitCode(errors.New("not signed in; run twinbay auth login"), clierrors.ExitAuth)

type Session struct {
	AccessToken    string `json:"access_token"`
	RefreshToken   string `json:"refresh_token"`
	OrganizationID string `json:"organization_id,omitempty"`
	User           User   `json:"user"`
}

type TokenClaims struct {
	ExpiresAt int64  `json:"exp"`
	SessionID string `json:"sid"`
}

func ParseTokenClaims(token string) (TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return TokenClaims{}, errors.New("invalid access token")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return TokenClaims{}, errors.New("invalid access token")
	}
	var claims TokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return TokenClaims{}, errors.New("invalid access token")
	}
	return claims, nil
}

type Store interface {
	Load() (*Session, error)
	Save(*Session) error
	Delete() error
}

type KeyringStore struct{}

func (KeyringStore) Load() (*Session, error) {
	raw := config.GetKeyringValue(sessionKey)
	if raw == "" {
		return nil, ErrNoSession
	}
	var session Session
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return nil, errors.New("stored browser session is invalid; run twinbay auth login again")
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		return nil, errors.New("stored browser session is incomplete; run twinbay auth login again")
	}
	return &session, nil
}

func (KeyringStore) Save(session *Session) error {
	if !config.KeyringAvailable() {
		return errors.New("OS keychain is unavailable; browser sessions cannot be stored securely")
	}
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	if err := config.SetKeyringValue(sessionKey, string(raw)); err != nil {
		return fmt.Errorf("store browser session in OS keychain: %w", err)
	}
	return nil
}

func (KeyringStore) Delete() error {
	if !config.KeyringAvailable() {
		return nil
	}
	if err := config.DeleteKeyringValue(sessionKey); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	return nil
}

type Manager struct {
	Client *Client
	Store  Store
	Now    func() time.Time

	mu      sync.Mutex
	loaded  bool
	session *Session
}

func NewManager() *Manager {
	return &Manager{Client: NewClient(), Store: KeyringStore{}, Now: time.Now}
}

func (m *Manager) Session(ctx context.Context) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessionLocked(ctx)
}

func (m *Manager) sessionLocked(ctx context.Context) (*Session, error) {
	if !m.loaded {
		session, err := m.Store.Load()
		if err != nil {
			return nil, err
		}
		m.session, m.loaded = session, true
	}
	claims, err := ParseTokenClaims(m.session.AccessToken)
	if err == nil && claims.ExpiresAt > m.now().Add(30*time.Second).Unix() {
		copy := *m.session
		return &copy, nil
	}

	refreshed, err := m.Client.Refresh(ctx, m.session.RefreshToken, m.session.OrganizationID)
	if err != nil {
		var oauthErr *OAuthError
		if errors.As(err, &oauthErr) && oauthErr.Code == "invalid_grant" {
			_ = m.Store.Delete()
			m.session, m.loaded = nil, false
			return nil, authenticationError(errors.New("browser session expired; run twinbay auth login again"))
		}
		return nil, err
	}
	m.session = sessionFromTokens(refreshed)
	if err := m.Store.Save(m.session); err != nil {
		return nil, err
	}
	copy := *m.session
	return &copy, nil
}

func (m *Manager) AccessToken(ctx context.Context) (string, error) {
	session, err := m.Session(ctx)
	if err != nil {
		return "", err
	}
	return session.AccessToken, nil
}

func (m *Manager) Switch(ctx context.Context, organizationID string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.loaded {
		session, err := m.Store.Load()
		if err != nil {
			return nil, err
		}
		m.session, m.loaded = session, true
	}
	refreshed, err := m.Client.Refresh(ctx, m.session.RefreshToken, organizationID)
	if err != nil {
		return nil, err
	}
	m.session = sessionFromTokens(refreshed)
	if err := m.Store.Save(m.session); err != nil {
		return nil, err
	}
	copy := *m.session
	return &copy, nil
}

func (m *Manager) Save(tokens *TokenResponse) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.session, m.loaded = sessionFromTokens(tokens), true
	return m.Store.Save(m.session)
}

func (m *Manager) Logout(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	var loadErr error
	if !m.loaded {
		session, err := m.Store.Load()
		if err != nil && !errors.Is(err, ErrNoSession) {
			loadErr = err
		}
		m.session, m.loaded = session, true
	}
	var remoteErr error
	if m.session != nil {
		remoteErr = m.Client.Logout(ctx, m.session.AccessToken)
	}
	deleteErr := m.Store.Delete()
	m.session = nil
	m.loaded = false
	if deleteErr != nil {
		return fmt.Errorf("clear browser session: %w", deleteErr)
	}
	if loadErr != nil {
		return fmt.Errorf("browser session could not be read for remote termination; local session was cleared: %w", loadErr)
	}
	return remoteErr
}

func sessionFromTokens(tokens *TokenResponse) *Session {
	return &Session{
		AccessToken:    tokens.AccessToken,
		RefreshToken:   tokens.RefreshToken,
		OrganizationID: tokens.OrganizationID,
		User:           tokens.User,
	}
}

func (m *Manager) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}
