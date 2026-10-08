package cloud

import (
	"context"
	"errors"
	"net/http"
)

// StackAccessToken returns only a Cloud data-plane token. The source validates
// and refreshes its coordinated session exactly as it does for HTTP requests.
// Direct and Membership clients cannot expose a token through this helper.
func StackAccessToken(ctx context.Context, client *http.Client) (string, error) {
	if client == nil {
		return "", errors.New("stack token requires an authenticated Cloud stack client")
	}
	source, ok := client.Transport.(*sessionTransport)
	if !ok || source == nil {
		return "", errors.New("stack token requires an authenticated Cloud stack client")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	//nolint:contextcheck // Client lifetime also cancels independent token callers.
	stop := context.AfterFunc(source.ctx, cancel)
	defer stop()
	if err := source.ctx.Err(); err != nil {
		return "", err
	}
	token, err := source.token(ctx)
	if err != nil {
		return "", err
	}
	if !usable(token) {
		return "", errors.New("cloud stack returned an invalid token")
	}
	return token.AccessToken, nil
}
