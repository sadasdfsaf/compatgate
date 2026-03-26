package protocols

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func LoadSource(ctx context.Context, location string) ([]byte, error) {
	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
		if err != nil {
			return nil, err
		}
		client := &http.Client{Timeout: 20 * time.Second}
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode >= 400 {
			return nil, fmt.Errorf("failed to fetch %s: %s", location, res.Status)
		}
		return io.ReadAll(res.Body)
	}
	return os.ReadFile(location)
}
