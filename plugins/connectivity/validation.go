package connectivity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/formancehq/fctl/v4/pkg/pluginsdk"
)

var dnsLabel = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func validateRequest(op operation, req pluginsdk.ExecuteRequest) error {
	if err := validateIDs(req.Args); err != nil {
		return err
	}
	if op.list {
		n, err := strconv.ParseUint(req.Flags["page-size"], 10, 32)
		if err != nil || n < 1 || n > 100 {
			return fmt.Errorf("page-size must be between 1 and 100")
		}
	}
	if op.query && req.Flags["query"] != "" {
		if _, err := object([]byte(req.Flags["query"]), "query"); err != nil {
			return err
		}
	}
	switch op.method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return validateBody(op.method, req.Body)
	default:
		if req.Body != nil {
			return fmt.Errorf("this read/delete command does not accept a JSON body")
		}
	}
	return nil
}

func validateIDs(args []string) error {
	for i, arg := range args {
		if i == 0 && !dnsLabel.MatchString(arg) {
			return fmt.Errorf("resource name must contain lowercase letters, digits or hyphens and start and end with a letter or digit")
		}
		if strings.TrimSpace(arg) == "" || arg == "." || arg == ".." {
			return fmt.Errorf("resource identifier must be non-empty and cannot be a dot or double dot")
		}
	}
	return nil
}

func validateBody(method string, data json.RawMessage) error {
	if len(data) > 1<<20 {
		return fmt.Errorf("connectivity request body exceeds 1 MiB")
	}
	payload, err := object(data, "data")
	if err != nil {
		return err
	}
	if method == http.MethodPatch {
		return validateSpec(payload, true)
	}
	if err := validateMetadata(payload); err != nil {
		return err
	}
	spec, err := object(payload["spec"], "data.spec")
	if err != nil {
		return err
	}
	if err := validateSpec(spec, false); err != nil {
		return err
	}
	if method == http.MethodPost {
		return validateCreate(payload, spec)
	}
	return nil
}

func validateCreate(payload, spec map[string]json.RawMessage) error {
	var name string
	if json.Unmarshal(payload["name"], &name) != nil || !dnsLabel.MatchString(name) {
		return fmt.Errorf("data.name must be a lowercase DNS label")
	}
	for _, key := range []string{"connector", "ledger"} {
		var value string
		if json.Unmarshal(spec[key], &value) != nil || strings.TrimSpace(value) == "" {
			return fmt.Errorf("data.spec.%s is required for creation", key)
		}
	}
	return nil
}

func validateMetadata(payload map[string]json.RawMessage) error {
	for _, key := range []string{"labels", "annotations"} {
		if value, exists := payload[key]; exists {
			values, err := object(value, "data."+key)
			if err != nil {
				return err
			}
			for _, v := range values {
				if !isString(v) {
					return fmt.Errorf("data.%s values must be strings", key)
				}
			}
		}
	}
	return nil
}

func object(data []byte, field string) (map[string]json.RawMessage, error) {
	var result map[string]json.RawMessage
	if json.Unmarshal(data, &result) != nil || result == nil {
		return nil, fmt.Errorf("%s must be a JSON object", field)
	}
	return result, nil
}

func isString(value json.RawMessage) bool {
	return len(bytes.TrimSpace(value)) != 0 && bytes.TrimSpace(value)[0] == '"'
}

func validateSpec(spec map[string]json.RawMessage, patch bool) error {
	for key, value := range spec {
		value = bytes.TrimSpace(value)
		if patch && bytes.Equal(value, []byte("null")) {
			continue
		}
		if err := validateSpecField(key, value); err != nil {
			return err
		}
	}
	return nil
}

func validateSpecField(key string, value json.RawMessage) error {
	switch key {
	case "connector", "version", "channel", "ledger", "pollInterval", "connectivityRef":
		if !isString(value) {
			return fmt.Errorf("spec.%s must be a string", key)
		}
	case "suspend":
		if string(value) != "true" && string(value) != "false" {
			return fmt.Errorf("spec.suspend must be a boolean")
		}
	case "startSequence":
		return validateInteger(value, key, math.MaxInt64)
	case "replicas":
		return validateInteger(value, key, 1)
	case "config", "additionalEnv":
		_, err := object(value, "spec."+key)
		return err
	case "additionalFiles":
		var files []json.RawMessage
		if json.Unmarshal(value, &files) != nil || files == nil {
			return fmt.Errorf("spec.additionalFiles must be an array")
		}
	}
	return nil
}

func validateInteger(value json.RawMessage, key string, maximum int64) error {
	var n int64
	if json.Unmarshal(value, &n) != nil || string(value) == "null" || n < 0 || n > maximum {
		return fmt.Errorf("spec.%s must be an integer between 0 and %d", key, maximum)
	}
	return nil
}
