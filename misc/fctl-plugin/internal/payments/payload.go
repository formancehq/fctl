package payments

import (
	"encoding/json"
	"fmt"
	"slices"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func validatePaymentPayload(command string, body json.RawMessage, major, minor int) error {
	if body == nil {
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(body, &object); err != nil || object == nil {
		return fmt.Errorf("expected a JSON object")
	}
	for _, validate := range []func(map[string]json.RawMessage) error{validatePaymentNumbers, validatePaymentTimes, validatePaymentStrings, validatePaymentMetadata} {
		if err := validate(object); err != nil {
			return err
		}
	}
	if err := validatePaymentEnums(command, object); err != nil {
		return err
	}
	if command == "bank-accounts create" && major < 3 {
		if _, err := commandapi.ObjectBody(body, "country"); err != nil {
			return err
		}
	}
	if command == "pools create" || command == "pools update-query" {
		return validatePoolPayload(object, major, minor)
	}
	return nil
}

func rawEnum(object map[string]json.RawMessage, field string, allowed []string) error {
	var value string
	if json.Unmarshal(object[field], &value) != nil || !slices.Contains(allowed, value) {
		return fmt.Errorf("invalid payload %s", field)
	}
	return nil
}

func validatePaymentNumbers(object map[string]json.RawMessage) error {
	for _, field := range []string{"amount", "initialAmount"} {
		if raw, exists := object[field]; exists {
			if _, err := integer(string(raw), true); err != nil {
				return fmt.Errorf("%s: %w", field, err)
			}
		}
	}
	return nil
}

func validatePaymentTimes(object map[string]json.RawMessage) error {
	for _, field := range []string{"createdAt", "scheduledAt"} {
		if raw, exists := object[field]; exists {
			var value string
			if json.Unmarshal(raw, &value) != nil {
				return fmt.Errorf("%s must be an RFC3339 string", field)
			}
			if err := commandapi.Timestamp(value); err != nil {
				return err
			}
		}
	}
	return nil
}

func validatePaymentStrings(object map[string]json.RawMessage) error {
	for _, field := range []string{"name", "connectorID", "reference", "asset", "description", "sourceAccountID", "destinationAccountID", "scheme"} {
		if raw, exists := object[field]; exists && string(raw) != "null" {
			var value string
			if json.Unmarshal(raw, &value) != nil {
				return fmt.Errorf("%s must be a string", field)
			}
		}
	}
	return nil
}

func validatePaymentMetadata(object map[string]json.RawMessage) error {
	if raw, exists := object["metadata"]; exists && string(raw) != "null" {
		var values map[string]string
		if json.Unmarshal(raw, &values) != nil || values == nil {
			return fmt.Errorf("metadata must be an object with string values")
		}
	}
	if raw, exists := object["validated"]; exists {
		var value bool
		if json.Unmarshal(raw, &value) != nil {
			return fmt.Errorf("validated must be a boolean")
		}
	}
	return nil
}

func validatePaymentEnums(command string, object map[string]json.RawMessage) error {
	if command == "accounts create" {
		if err := rawEnum(object, "type", []string{"UNKNOWN", "INTERNAL", "EXTERNAL"}); err != nil {
			return err
		}
	}
	if command == "payments create" {
		if err := rawEnum(object, "type", []string{"PAY-IN", "PAYOUT", "TRANSFER", "OTHER"}); err != nil {
			return err
		}
		if err := rawEnum(object, "status", []string{"PENDING", "SUCCEEDED", "CANCELLED", "FAILED", "EXPIRED", "REFUNDED", "REFUNDED_FAILURE", "DISPUTE", "DISPUTE_WON", "DISPUTE_LOST", "OTHER"}); err != nil {
			return err
		}
	}
	if command == "transfer-initiation create" {
		if err := rawEnum(object, "type", []string{"TRANSFER", "PAYOUT"}); err != nil {
			return err
		}
	}
	return nil
}
