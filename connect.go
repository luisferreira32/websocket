package websocket

import (
	"context"
	"net/http"
	neturl "net/url"
)

type Connection interface{}

func Connect(ctx context.Context, url string) (Connection, error) {
	// TODO: RFC6455 Section 3. validation
	u, err := neturl.Parse(url)
	if err != nil {
		return nil, err
	}

	// TODO: the rest!
	upgradeRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, err
	}
	upgradeRequest.Header.Add("Upgrade", "Websocket")
	return nil, nil
}
