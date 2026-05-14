package capabilities

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var versionedTagPattern = regexp.MustCompile(`^([A-Za-z0-9_-]+)\.(v[0-9]+)$`)
var pathVersionPattern = regexp.MustCompile(`^/api/([^/]+)/((?:v)[0-9]+)(?:/|$)`)

func ParseOpenAPIManifest(reader io.Reader) (Manifest, error) {
	var document openAPIDocument
	if err := json.NewDecoder(reader).Decode(&document); err != nil {
		return Manifest{}, fmt.Errorf("decode openapi document: %w", err)
	}

	manifest := Manifest{
		SpecVersion: document.Info.Version,
		Products:    map[Product]ProductManifest{},
	}

	paths := make([]string, 0, len(document.Paths))
	for path := range document.Paths {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		pathItem := document.Paths[path]
		methods := make([]string, 0, len(pathItem))
		for method := range pathItem {
			methods = append(methods, method)
		}
		sort.Strings(methods)

		for _, method := range methods {
			if !isHTTPMethod(method) {
				continue
			}
			var operation openAPIOperation
			if err := json.Unmarshal(pathItem[method], &operation); err != nil {
				return Manifest{}, fmt.Errorf("decode operation %s %s: %w", method, path, err)
			}
			if operation.OperationID == "" {
				continue
			}

			product, apiVersion, ok := operationProductVersion(path, operation.Tags)
			if !ok {
				continue
			}

			productManifest := manifest.Products[product]
			if productManifest.Operations == nil {
				productManifest.Operations = map[Feature]map[APIVersion]Operation{}
			}
			productManifest.APIVersions = append(productManifest.APIVersions, apiVersion)

			feature := Feature(canonicalFeature(operation.OperationID))
			if productManifest.Operations[feature] == nil {
				productManifest.Operations[feature] = map[APIVersion]Operation{}
			}
			productManifest.Operations[feature][apiVersion] = Operation{
				OperationID: operation.OperationID,
				Method:      strings.ToUpper(method),
				Path:        path,
				Tags:        append([]string(nil), operation.Tags...),
			}
			manifest.Products[product] = productManifest
		}
	}

	for product, productManifest := range manifest.Products {
		productManifest.APIVersions = UniqueSortedAPIVersions(productManifest.APIVersions)
		manifest.Products[product] = productManifest
	}

	return manifest, nil
}

func operationProductVersion(path string, tags []string) (Product, APIVersion, bool) {
	for _, tag := range tags {
		matches := versionedTagPattern.FindStringSubmatch(tag)
		if len(matches) == 3 {
			return Product(matches[1]), APIVersion(matches[2]), true
		}
	}

	matches := pathVersionPattern.FindStringSubmatch(path)
	if len(matches) == 3 {
		return Product(matches[1]), APIVersion(matches[2]), true
	}

	return "", "", false
}

func canonicalFeature(operationID string) string {
	for {
		if len(operationID) < 3 || operationID[0] != 'v' {
			break
		}
		i := 1
		for i < len(operationID) && operationID[i] >= '0' && operationID[i] <= '9' {
			i++
		}
		if i == 1 || i >= len(operationID) {
			break
		}
		operationID = operationID[i:]
		break
	}
	if operationID == "" {
		return operationID
	}
	r, size := utf8.DecodeRuneInString(operationID)
	if r == utf8.RuneError {
		return operationID
	}
	return string(unicode.ToLower(r)) + operationID[size:]
}

type openAPIDocument struct {
	Info  openAPIInfo                           `json:"info"`
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

type openAPIInfo struct {
	Version string `json:"version"`
}

type openAPIOperation struct {
	OperationID string   `json:"operationId"`
	Tags        []string `json:"tags"`
}

func isHTTPMethod(method string) bool {
	switch strings.ToLower(method) {
	case "get", "put", "post", "delete", "options", "head", "patch", "trace":
		return true
	default:
		return false
	}
}
