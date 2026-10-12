package command

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/fctl/pkg/pluginsdk"
	"github.com/formancehq/fctl/pkg/pluginsdk/httpclient"
)

func Perform(ctx context.Context, client *httpclient.Client, r pluginsdk.ExecuteRequest, op Operation, body json.RawMessage) (json.RawMessage, error) {
	path, err := Route(op.Segments, r)
	if err != nil {
		return nil, err
	}
	query, err := QueryValues(op.Query, r)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	key := r.Flags["idempotency-key"]
	if key == "" {
		key = r.Flags["ik"]
	}
	if key != "" {
		headers.Set("Idempotency-Key", key)
	}
	return client.Do(ctx, op.Method, path, query, body, headers)
}

func Identifier(value string) error {
	if strings.TrimSpace(value) == "" || value == "." || value == ".." {
		return fmt.Errorf("identifier must be nonempty and cannot be a dot segment")
	}
	return nil
}

func Route(segments []string, r pluginsdk.ExecuteRequest) (string, error) {
	resolved := make([]string, len(segments))
	for i, segment := range segments {
		value := segment
		if strings.HasPrefix(segment, "$") {
			n, err := strconv.Atoi(segment[1:])
			if err != nil || n < 0 || n >= len(r.Args) {
				return "", fmt.Errorf("missing route argument %s", segment)
			}
			value = r.Args[n]
		} else if strings.HasPrefix(segment, "@") {
			value = r.Flags[segment[1:]]
		}
		if err := Identifier(value); err != nil {
			return "", fmt.Errorf("%s: %w", segment, err)
		}
		resolved[i] = value
	}
	return httpclient.Path(resolved...), nil
}

func QueryValues(bindings map[string]string, r pluginsdk.ExecuteRequest) (url.Values, error) {
	query := url.Values{}
	for flag, wire := range bindings {
		value := r.Flags[flag]
		if value == "" {
			continue
		}
		if flag == "page-size" {
			n, err := strconv.ParseUint(value, 10, 32)
			if err != nil || n == 0 || n > 1000 {
				return nil, fmt.Errorf("--page-size must be between 1 and 1000")
			}
		}
		if strings.Contains(flag, "created-at") || flag == "at" || flag == "timestamp" {
			if err := Timestamp(value); err != nil {
				return nil, fmt.Errorf("--%s: %w", flag, err)
			}
		}
		query.Set(wire, value)
	}
	return query, nil
}

func Timestamp(value string) error {
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		return fmt.Errorf("expected RFC3339 timestamp: %w", err)
	}
	return nil
}
