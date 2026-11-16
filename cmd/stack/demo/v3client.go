package demo

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/formancehq/go-libs/v4/pointer"

	formance "github.com/formancehq/fctl/v3/pkg/client"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledger"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/ledgerv3"
	"github.com/formancehq/fctl/v3/pkg/client/pkg/models/operations"
)

const v3ScriptVersion = "1.0.0"

// v3LedgerClient adapts the recipe runner's requests to Ledger V3. V3 uses
// account types and a versioned Numscript library instead of V2 schemas.
type v3LedgerClient struct {
	client *formance.V3
	schema ledger.V2SchemaDataInput
}

func (c *v3LedgerClient) CreateLedger(ctx context.Context, request operations.V2CreateLedgerRequest, opts ...operations.Option) (*operations.V2CreateLedgerResponse, error) {
	response, err := c.client.CreateLedger(ctx, operations.V3CreateLedgerRequest{
		LedgerName:     request.Ledger,
		IdempotencyKey: pointer.For(request.Ledger + ":create"),
		CreateLedgerRequest: ledgerv3.CreateLedgerRequest{
			AccountTypes:           v3AccountTypes(c.schema.V2ChartOfAccounts),
			DefaultEnforcementMode: ledgerv3.DefaultEnforcementModeStrict.ToPointer(),
			Metadata:               v3Metadata(request.V2CreateLedgerRequest.V2Metadata),
			Mode:                   ledgerv3.ModeNormal.ToPointer(),
		},
	}, nil, opts...)
	if err != nil {
		return nil, err
	}
	if err := v3ResponseError(response.StatusCode, response.ErrorResponse); err != nil {
		return nil, err
	}
	return &operations.V2CreateLedgerResponse{StatusCode: http.StatusNoContent}, nil
}

func (c *v3LedgerClient) InsertSchema(ctx context.Context, request operations.V2InsertSchemaRequest, opts ...operations.Option) (*operations.V2InsertSchemaResponse, error) {
	for _, name := range slices.Sorted(maps.Keys(request.V2SchemaData.V2TransactionTemplates)) {
		template := request.V2SchemaData.V2TransactionTemplates[name]
		saveOpts := append(slices.Clone(opts), operations.WithSetHeaders(map[string]string{"Idempotency-Key": request.Ledger + ":" + name + ":" + v3ScriptVersion}))
		response, err := c.client.SaveNumscript(ctx, operations.V3SaveNumscriptRequest{
			LedgerName:           request.Ledger,
			Name:                 name,
			SaveNumscriptRequest: ledgerv3.SaveNumscriptRequest{Content: template.Script, Version: v3ScriptVersion},
		}, nil, saveOpts...)
		if err != nil {
			return nil, fmt.Errorf("save recipe script %s: %w", name, err)
		}
		if err := v3ResponseError(response.StatusCode, response.ErrorResponse); err != nil {
			return nil, fmt.Errorf("save recipe script %s: %w", name, err)
		}
	}
	return &operations.V2InsertSchemaResponse{StatusCode: http.StatusNoContent}, nil
}

func (c *v3LedgerClient) CreateTransaction(ctx context.Context, request operations.V2CreateTransactionRequest, opts ...operations.Option) (*operations.V2CreateTransactionResponse, error) {
	script := request.V2PostTransaction.Script
	if script == nil || (script.Template == nil) == (script.Plain == nil) {
		return nil, fmt.Errorf("demo transaction requires exactly one script template or inline script")
	}
	body := ledgerv3.CreateTransactionRequest{
		Reference: request.V2PostTransaction.Reference,
		Metadata:  v3Metadata(request.V2PostTransaction.V2Metadata),
		Timestamp: request.V2PostTransaction.Timestamp,
	}
	var source string
	if script.Plain != nil {
		source = *script.Plain
		body.Script = &ledgerv3.Script{Plain: source, Vars: script.Vars}
	} else {
		template, ok := c.schema.V2TransactionTemplates[*script.Template]
		if !ok {
			return nil, fmt.Errorf("unknown recipe script %q", *script.Template)
		}
		source = template.Script
		body.ScriptReference = &ledgerv3.ScriptReference{Name: *script.Template, Version: v3ScriptVersion, Vars: script.Vars}
	}
	body.AccountMetadata = v3RecipeAccountMetadata(c.schema, source, script.Vars)
	for account, metadata := range request.V2PostTransaction.AccountMetadata {
		if body.AccountMetadata[account] == nil {
			body.AccountMetadata[account] = make(map[string]*ledgerv3.MetadataValue)
		}
		maps.Copy(body.AccountMetadata[account], v3Metadata(metadata))
	}
	response, err := c.client.CreateTransaction(ctx, operations.V3CreateTransactionRequest{
		LedgerName:     request.Ledger,
		IdempotencyKey: request.IdempotencyKey,
		RequestBody:    operations.CreateV3CreateTransactionRequestBodyCreateTransactionRequest(body),
	}, nil, opts...)
	if err != nil {
		return nil, err
	}
	if err := v3ResponseError(response.StatusCode, response.ErrorResponse); err != nil {
		return nil, err
	}
	return &operations.V2CreateTransactionResponse{StatusCode: http.StatusOK}, nil
}

func v3Metadata(metadata map[string]string) map[string]*ledgerv3.MetadataValue {
	result := make(map[string]*ledgerv3.MetadataValue, len(metadata))
	for key, value := range metadata {
		result[key] = pointer.For(ledgerv3.CreateMetadataValueStr(value))
	}
	return result
}

var scriptAccountLiteral = regexp.MustCompile(`@[a-zA-Z0-9_$:-]+`)

// Account defaults are applied only to concrete literals referenced by this
// script. Catalog validation evaluates every step and checks these addresses
// are involved in its postings, including refund and reversal branches.
func v3RecipeAccountMetadata(schema ledger.V2SchemaDataInput, source string, vars map[string]string) map[string]map[string]*ledgerv3.MetadataValue {
	result := make(map[string]map[string]*ledgerv3.MetadataValue)
	for _, literal := range scriptAccountLiteral.FindAllString(source, -1) {
		parts := strings.Split(strings.TrimPrefix(literal, "@"), ":")
		resolved := true
		for i, part := range parts {
			if variable, ok := strings.CutPrefix(part, "$"); ok {
				value, exists := vars[variable]
				if !exists || value == "" {
					resolved = false
					break
				}
				parts[i] = value
			}
		}
		if !resolved {
			continue
		}
		account := strings.Join(parts, ":")
		defaults := chartAccountDefaults(schema.V2ChartOfAccounts, strings.Split(account, ":"))
		if len(defaults) > 0 {
			result[account] = v3Metadata(defaults)
		}
	}
	return result
}

func chartAccountDefaults(chart map[string]ledger.V2ChartSegment, parts []string) map[string]string {
	if len(parts) == 0 {
		return nil
	}
	for _, name := range slices.Sorted(maps.Keys(chart)) {
		segment := chart[name]
		if name != parts[0] {
			if !strings.HasPrefix(name, "$") {
				continue
			}
			if segment.DotPattern != nil {
				matched, err := regexp.MatchString(*segment.DotPattern, parts[0])
				if err != nil || !matched {
					continue
				}
			}
		}
		if len(parts) > 1 {
			if defaults := chartAccountDefaults(segment.AdditionalProperties, parts[1:]); len(defaults) > 0 {
				return defaults
			}
			continue
		}
		defaults := make(map[string]string)
		for key, metadata := range segment.DotMetadata {
			if metadata.Default != nil {
				defaults[key] = *metadata.Default
			}
		}
		return defaults
	}
	return nil
}

func v3AccountTypes(chart map[string]ledger.V2ChartSegment) map[string]ledgerv3.AddAccountTypeRequest {
	result := make(map[string]ledgerv3.AddAccountTypeRequest)
	var walk func(map[string]ledger.V2ChartSegment, []string, map[string]*ledgerv3.SegmentType)
	walk = func(segments map[string]ledger.V2ChartSegment, path []string, constraints map[string]*ledgerv3.SegmentType) {
		for _, name := range slices.Sorted(maps.Keys(segments)) {
			segment := segments[name]
			nextConstraints := maps.Clone(constraints)
			if variable, ok := strings.CutPrefix(name, "$"); ok {
				name = "{" + variable + "}"
				if segment.DotPattern != nil {
					nextConstraints[variable] = &ledgerv3.SegmentType{Type: ledgerv3.SegmentTypeTypeRegex, Regex: segment.DotPattern}
				}
			}
			nextPath := append(slices.Clone(path), name)
			if segment.DotSelf != nil || len(segment.AdditionalProperties) == 0 {
				pattern := strings.Join(nextPath, ":")
				typeName := strings.NewReplacer(":", "_", "{", "", "}", "").Replace(pattern)
				result[typeName] = ledgerv3.AddAccountTypeRequest{Name: typeName, Pattern: pattern, Persistence: ledgerv3.AddAccountTypeRequestPersistenceNormal.ToPointer(), SegmentTypes: nextConstraints}
			}
			walk(segment.AdditionalProperties, nextPath, nextConstraints)
		}
	}
	walk(chart, nil, map[string]*ledgerv3.SegmentType{})
	return result
}

type v3APIError struct {
	StatusCode int
	Response   *ledgerv3.ErrorResponse
}

func (e *v3APIError) Error() string {
	if e.Response != nil {
		return fmt.Sprintf("Ledger V3 HTTP %d: %s: %s", e.StatusCode, e.Response.ErrorCode, e.Response.ErrorMessage)
	}
	return fmt.Sprintf("Ledger V3 HTTP %d", e.StatusCode)
}
func v3ResponseError(status int, response *ledgerv3.ErrorResponse) error {
	if status < 200 || status >= 300 {
		return &v3APIError{StatusCode: status, Response: response}
	}
	return nil
}
