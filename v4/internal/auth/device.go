package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/fctl/v4/internal/credentials"
)

const DeviceClientID = "fctl"

var ErrOpeningBrowser = errors.New("opening browser")

type DeviceLoginOptions struct {
	IssuerURL      string
	ClientID       string
	Scopes         []string
	Resources      []string
	Prompt         []string
	OrganizationID string
	IDTokenHint    string
	HTTPClient     *http.Client
	OpenURL        func(string) error
	Out            io.Writer
}

type DeviceTokens struct {
	AccessToken  StoredAccessToken `json:"accessToken"`
	IDToken      string            `json:"idToken,omitempty"`
	RefreshToken string            `json:"refreshToken,omitempty"`
}

type StoredAccessToken struct {
	Token        string    `json:"token"`
	TokenType    string    `json:"tokenType,omitempty"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

func MarshalDeviceTokens(tokens DeviceTokens) (string, error) {
	encoded, err := json.Marshal(tokens)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func ParseDeviceTokens(value string) (DeviceTokens, error) {
	return parseStoredDeviceTokens(value)
}

func DeviceLogin(ctx context.Context, options DeviceLoginOptions) (DeviceTokens, error) {
	if options.IssuerURL == "" {
		return DeviceTokens{}, errors.New("issuer URL is required")
	}
	clientID := options.ClientID
	if clientID == "" {
		clientID = DeviceClientID
	}
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	discovery, err := discoverOIDC(ctx, httpClient, options.IssuerURL)
	if err != nil {
		return DeviceTokens{}, err
	}
	if discovery.DeviceAuthorizationEndpoint == "" {
		return DeviceTokens{}, errors.New("oidc discovery did not include device_authorization_endpoint")
	}

	deviceCode, err := initiateDeviceAuthorization(ctx, httpClient, discovery.DeviceAuthorizationEndpoint, clientID, options)
	if err != nil {
		return DeviceTokens{}, fmt.Errorf("failed to initiate device authorization flow: %w", err)
	}

	verificationURL := deviceVerificationURL(deviceCode)
	openURL := options.OpenURL
	if openURL == nil {
		openURL = OpenURL
	}
	if err := openURL(verificationURL); err != nil {
		if !errors.Is(err, ErrOpeningBrowser) {
			return DeviceTokens{}, err
		}
		writeDeviceMessage(options.Out, "No browser detected")
		writeDeviceMessage(options.Out, "Please open the following URL in your browser: "+verificationURL)
	} else {
		writeDeviceMessage(options.Out, "A browser window has been opened on "+verificationURL)
	}
	writeDeviceMessage(options.Out, "Waiting for authentication...")

	token, err := pollDeviceToken(ctx, httpClient, discovery.TokenEndpoint, clientID, deviceCode)
	if err != nil {
		return DeviceTokens{}, fmt.Errorf("failed to get access token: %w", err)
	}
	return token, nil
}

func OpenURL(urlString string) error {
	var cmd string
	var args []string

	if _, err := url.Parse(urlString); err != nil {
		return fmt.Errorf("invalid URL: %s", urlString)
	}

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default:
		cmd = "xdg-open"
	}

	if _, err := exec.LookPath(cmd); err == nil {
		args = append(args, urlString)
		return exec.Command(cmd, args...).Start() //nolint:gosec
	}
	return ErrOpeningBrowser
}

type DeviceTokenSource struct {
	IssuerURL  string
	TokenRef   string
	Scopes     []string
	Store      credentials.Store
	HTTPClient *http.Client
	ClientID   string
	now        func() time.Time
}

func (s *DeviceTokenSource) Token(ctx context.Context) (Token, error) {
	if s.Store == nil {
		return Token{}, errors.New("credential store is required")
	}
	if s.TokenRef == "" {
		return Token{}, errors.New("tokenRef is required for device auth")
	}
	value, err := s.Store.Get(ctx, s.TokenRef)
	if err != nil {
		return Token{}, err
	}
	tokens, err := parseStoredDeviceTokens(value)
	if err != nil {
		return Token{}, err
	}
	if tokens.AccessToken.Token == "" {
		return Token{}, errors.New("stored device credentials do not include an access token")
	}
	if !s.tokenExpired(tokens.AccessToken) && s.tokenHasScopes(tokens.AccessToken) {
		return normalizeToken(Token{AccessToken: tokens.AccessToken.Token, TokenType: tokens.AccessToken.TokenType}), nil
	}
	if tokens.AccessToken.RefreshToken == "" && tokens.RefreshToken == "" {
		return Token{}, errors.New("device access token is expired or missing required scopes; run `fctl login` again")
	}
	refreshed, err := s.refresh(ctx, tokens)
	if err != nil {
		return Token{}, err
	}
	encoded, err := json.Marshal(refreshed)
	if err != nil {
		return Token{}, err
	}
	if err := s.Store.Set(ctx, s.TokenRef, string(encoded)); err != nil {
		return Token{}, err
	}
	return normalizeToken(Token{AccessToken: refreshed.AccessToken.Token, TokenType: refreshed.AccessToken.TokenType}), nil
}

func (s *DeviceTokenSource) tokenExpired(token StoredAccessToken) bool {
	if token.Expiry.IsZero() {
		if expiry, ok := tokenExpiry(token.Token); ok {
			return !expiry.After(s.currentTime().Add(30 * time.Second))
		}
		return false
	}
	return !token.Expiry.After(s.currentTime().Add(30 * time.Second))
}

func (s *DeviceTokenSource) tokenHasScopes(token StoredAccessToken) bool {
	if len(s.Scopes) == 0 {
		return true
	}
	claims, ok := jwtClaims(token.Token)
	if !ok {
		return true
	}
	available := map[string]struct{}{}
	switch value := claims["scope"].(type) {
	case string:
		for _, scope := range strings.Fields(value) {
			available[scope] = struct{}{}
		}
	case []any:
		for _, scope := range value {
			if text, ok := scope.(string); ok {
				available[text] = struct{}{}
			}
		}
	}
	for _, scope := range s.Scopes {
		if _, ok := available[scope]; !ok {
			return false
		}
	}
	return true
}

func (s *DeviceTokenSource) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *DeviceTokenSource) refresh(ctx context.Context, tokens DeviceTokens) (DeviceTokens, error) {
	if s.IssuerURL == "" {
		return DeviceTokens{}, errors.New("issuer URL is required to refresh device auth")
	}
	httpClient := s.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	clientID := s.ClientID
	if clientID == "" {
		clientID = DeviceClientID
	}
	discovery, err := discoverOIDC(ctx, httpClient, s.IssuerURL)
	if err != nil {
		return DeviceTokens{}, err
	}
	refreshToken := tokens.AccessToken.RefreshToken
	if refreshToken == "" {
		refreshToken = tokens.RefreshToken
	}
	next, err := refreshDeviceToken(ctx, httpClient, discovery.TokenEndpoint, clientID, refreshToken, s.Scopes)
	if err != nil {
		return DeviceTokens{}, err
	}
	if next.AccessToken.RefreshToken == "" {
		next.AccessToken.RefreshToken = refreshToken
	}
	if next.RefreshToken == "" {
		next.RefreshToken = next.AccessToken.RefreshToken
	}
	if next.IDToken == "" {
		next.IDToken = tokens.IDToken
	}
	return next, nil
}

type oidcDiscovery struct {
	TokenEndpoint               string `json:"token_endpoint"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
}

func discoverOIDC(ctx context.Context, httpClient *http.Client, issuerURL string) (oidcDiscovery, error) {
	discoveryURL := strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return oidcDiscovery{}, err
	}
	rsp, err := httpClient.Do(req)
	if err != nil {
		return oidcDiscovery{}, err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(rsp.Body, 4096))
		return oidcDiscovery{}, fmt.Errorf("oidc discovery failed: status %d: %s", rsp.StatusCode, string(body))
	}
	var discovery oidcDiscovery
	if err := json.NewDecoder(rsp.Body).Decode(&discovery); err != nil {
		return oidcDiscovery{}, fmt.Errorf("decode oidc discovery: %w", err)
	}
	if discovery.TokenEndpoint == "" {
		return oidcDiscovery{}, errors.New("oidc discovery did not include token_endpoint")
	}
	return discovery, nil
}

type deviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	Interval                int    `json:"interval"`
	ExpiresIn               int    `json:"expires_in"`
	Resources               []string
}

func initiateDeviceAuthorization(ctx context.Context, httpClient *http.Client, endpoint string, clientID string, options DeviceLoginOptions) (deviceAuthorizationResponse, error) {
	form := url.Values{}
	if len(options.Scopes) > 0 {
		form.Set("scope", strings.Join(options.Scopes, " "))
	}
	if len(options.Prompt) > 0 {
		form.Set("prompt", strings.Join(options.Prompt, " "))
	}
	if options.OrganizationID != "" {
		form.Set("organization_id", options.OrganizationID)
	}
	if options.IDTokenHint != "" {
		form.Set("id_token_hint", options.IDTokenHint)
	}
	for _, resource := range options.Resources {
		form.Add("resource", resource)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return deviceAuthorizationResponse{}, err
	}
	req.SetBasicAuth(clientID, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rsp, err := httpClient.Do(req)
	if err != nil {
		return deviceAuthorizationResponse{}, err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(rsp.Body, 4096))
		return deviceAuthorizationResponse{}, fmt.Errorf("device authorization request failed: status %d: %s", rsp.StatusCode, string(body))
	}
	var output deviceAuthorizationResponse
	if err := json.NewDecoder(rsp.Body).Decode(&output); err != nil {
		return deviceAuthorizationResponse{}, fmt.Errorf("decode device authorization response: %w", err)
	}
	if output.DeviceCode == "" {
		return deviceAuthorizationResponse{}, errors.New("device authorization response did not include device_code")
	}
	output.Resources = append([]string(nil), options.Resources...)
	return output, nil
}

func deviceVerificationURL(deviceCode deviceAuthorizationResponse) string {
	if deviceCode.VerificationURI == "" {
		return deviceCode.VerificationURIComplete
	}
	uri, err := url.Parse(deviceCode.VerificationURI)
	if err != nil {
		return deviceCode.VerificationURI
	}
	query := uri.Query()
	query.Set("user_code", deviceCode.UserCode)
	uri.RawQuery = query.Encode()
	return uri.String()
}

func pollDeviceToken(ctx context.Context, httpClient *http.Client, endpoint string, clientID string, deviceCode deviceAuthorizationResponse) (DeviceTokens, error) {
	interval := time.Duration(deviceCode.Interval) * time.Second
	if interval <= 0 {
		interval = 5 * time.Second
	}
	deadline := time.Time{}
	if deviceCode.ExpiresIn > 0 {
		deadline = time.Now().Add(time.Duration(deviceCode.ExpiresIn) * time.Second)
	}
	for {
		form := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {deviceCode.DeviceCode},
		}
		for _, resource := range deviceCode.Resources {
			form.Add("resource", resource)
		}
		tokens, retry, nextInterval, err := requestDeviceToken(ctx, httpClient, endpoint, clientID, form, interval)
		if err == nil {
			return tokens, nil
		}
		if !retry {
			return DeviceTokens{}, err
		}
		interval = nextInterval
		if !deadline.IsZero() && time.Now().Add(interval).After(deadline) {
			return DeviceTokens{}, errors.New("device authorization expired")
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return DeviceTokens{}, ctx.Err()
		case <-timer.C:
		}
	}
}

func refreshDeviceToken(ctx context.Context, httpClient *http.Client, endpoint string, clientID string, refreshToken string, scopes []string) (DeviceTokens, error) {
	if refreshToken == "" {
		return DeviceTokens{}, errors.New("refresh token is required")
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	}
	if len(scopes) > 0 {
		form.Set("scope", strings.Join(scopes, " "))
	}
	tokens, _, _, err := requestDeviceToken(ctx, httpClient, endpoint, clientID, form, 0)
	return tokens, err
}

func requestDeviceToken(ctx context.Context, httpClient *http.Client, endpoint string, clientID string, form url.Values, interval time.Duration) (DeviceTokens, bool, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return DeviceTokens{}, false, interval, err
	}
	req.SetBasicAuth(clientID, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rsp, err := httpClient.Do(req)
	if err != nil {
		return DeviceTokens{}, false, interval, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode >= 200 && rsp.StatusCode < 300 {
		var tokenResponse struct {
			AccessToken  string `json:"access_token"`
			TokenType    string `json:"token_type"`
			RefreshToken string `json:"refresh_token"`
			IDToken      string `json:"id_token"`
			ExpiresIn    int64  `json:"expires_in"`
		}
		if err := json.NewDecoder(rsp.Body).Decode(&tokenResponse); err != nil {
			return DeviceTokens{}, false, interval, fmt.Errorf("decode token response: %w", err)
		}
		if tokenResponse.AccessToken == "" {
			return DeviceTokens{}, false, interval, errors.New("token response did not include access_token")
		}
		expiry := time.Time{}
		if tokenResponse.ExpiresIn > 0 {
			expiry = time.Now().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
		} else if jwtExpiry, ok := tokenExpiry(tokenResponse.AccessToken); ok {
			expiry = jwtExpiry
		}
		return DeviceTokens{
			AccessToken: StoredAccessToken{
				Token:        tokenResponse.AccessToken,
				TokenType:    tokenResponse.TokenType,
				RefreshToken: tokenResponse.RefreshToken,
				Expiry:       expiry,
			},
			IDToken:      tokenResponse.IDToken,
			RefreshToken: tokenResponse.RefreshToken,
		}, false, interval, nil
	}

	var tokenError struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	body, _ := io.ReadAll(io.LimitReader(rsp.Body, 4096))
	_ = json.Unmarshal(body, &tokenError)
	switch tokenError.Error {
	case "authorization_pending":
		return DeviceTokens{}, true, interval, errors.New(tokenError.Error)
	case "slow_down":
		return DeviceTokens{}, true, interval + 5*time.Second, errors.New(tokenError.Error)
	case "access_denied", "expired_token":
		if tokenError.ErrorDescription != "" {
			return DeviceTokens{}, false, interval, fmt.Errorf("%s: %s", tokenError.Error, tokenError.ErrorDescription)
		}
		return DeviceTokens{}, false, interval, errors.New(tokenError.Error)
	default:
		if tokenError.Error != "" {
			if tokenError.ErrorDescription != "" {
				return DeviceTokens{}, false, interval, fmt.Errorf("%s: %s", tokenError.Error, tokenError.ErrorDescription)
			}
			return DeviceTokens{}, false, interval, errors.New(tokenError.Error)
		}
		return DeviceTokens{}, false, interval, fmt.Errorf("token request failed: status %d: %s", rsp.StatusCode, string(body))
	}
}

func parseStoredDeviceTokens(value string) (DeviceTokens, error) {
	var tokens DeviceTokens
	if err := json.Unmarshal([]byte(value), &tokens); err == nil && tokens.AccessToken.Token != "" {
		return tokens, nil
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		return DeviceTokens{AccessToken: StoredAccessToken{Token: strings.TrimSpace(value), TokenType: "Bearer"}}, nil
	}
	access := objectValue(raw, "accessToken")
	if len(access) == 0 {
		access = objectValue(raw, "access")
	}
	id := objectValue(raw, "idToken")
	if len(id) == 0 {
		id = objectValue(raw, "id")
	}

	tokens.AccessToken.Token = stringValue(access, "token")
	tokens.AccessToken.TokenType = stringValue(access, "tokenType")
	tokens.AccessToken.RefreshToken = stringValue(access, "refreshToken")
	tokens.RefreshToken = tokens.AccessToken.RefreshToken
	tokens.IDToken = stringValue(id, "token")
	if expiry, ok := tokenExpiry(tokens.AccessToken.Token); ok {
		tokens.AccessToken.Expiry = expiry
	}
	if tokens.AccessToken.Token == "" {
		return DeviceTokens{}, errors.New("stored device credentials do not include an access token")
	}
	return tokens, nil
}

func objectValue(values map[string]any, key string) map[string]any {
	value, _ := values[key].(map[string]any)
	return value
}

func stringValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func tokenExpiry(token string) (time.Time, bool) {
	claims, ok := jwtClaims(token)
	if !ok {
		return time.Time{}, false
	}
	exp, ok := claims["exp"]
	if !ok {
		return time.Time{}, false
	}
	switch value := exp.(type) {
	case float64:
		return time.Unix(int64(value), 0), true
	case json.Number:
		seconds, err := strconv.ParseInt(value.String(), 10, 64)
		return time.Unix(seconds, 0), err == nil
	default:
		return time.Time{}, false
	}
}

func EmailFromIDToken(token string) string {
	claims, ok := jwtClaims(token)
	if !ok {
		return ""
	}
	email, _ := claims["email"].(string)
	return email
}

func jwtClaims(token string) (map[string]any, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	var claims map[string]any
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	if err := decoder.Decode(&claims); err != nil {
		return nil, false
	}
	return claims, true
}

func writeDeviceMessage(out io.Writer, message string) {
	if out == nil {
		return
	}
	fmt.Fprintln(out, message)
}
