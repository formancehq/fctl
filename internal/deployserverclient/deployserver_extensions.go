package deployserverclient

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/formancehq/fctl/internal/deployserverclient/v3/internal/hooks"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/internal/utils"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/apierrors"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/components"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/models/operations"
	"github.com/formancehq/fctl/internal/deployserverclient/v3/retry"
)

// doRequest is a shared helper that handles the full request lifecycle (hooks, retries, error responses).
func (s *DeployServer) doRequest(ctx context.Context, hookCtx hooks.HookContext, req *http.Request, o operations.Options) (*http.Response, error) {
	globalRetryConfig := s.sdkConfiguration.RetryConfig
	retryConfig := o.Retries
	if retryConfig == nil {
		if globalRetryConfig != nil {
			retryConfig = globalRetryConfig
		}
	}

	var httpRes *http.Response
	var err error

	if retryConfig != nil {
		httpRes, err = utils.Retry(ctx, utils.Retries{
			Config: retryConfig,
			StatusCodes: []string{
				"429", "500", "502", "503", "504",
			},
		}, func() (*http.Response, error) {
			if req.Body != nil && req.Body != http.NoBody && req.GetBody != nil {
				copyBody, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				req.Body = copyBody
			}

			req, err = s.hooks.BeforeRequest(hooks.BeforeRequestContext{HookContext: hookCtx}, req)
			if err != nil {
				if retry.IsPermanentError(err) || retry.IsTemporaryError(err) {
					return nil, err
				}
				return nil, retry.Permanent(err)
			}

			httpRes, err := s.sdkConfiguration.Client.Do(req)
			if err != nil || httpRes == nil {
				if err != nil {
					err = fmt.Errorf("error sending request: %w", err)
				} else {
					err = fmt.Errorf("error sending request: no response")
				}
				_, err = s.hooks.AfterError(hooks.AfterErrorContext{HookContext: hookCtx}, nil, err)
			}
			return httpRes, err
		})

		if err != nil {
			return nil, err
		}
		httpRes, err = s.hooks.AfterSuccess(hooks.AfterSuccessContext{HookContext: hookCtx}, httpRes)
		if err != nil {
			return nil, err
		}
	} else {
		req, err = s.hooks.BeforeRequest(hooks.BeforeRequestContext{HookContext: hookCtx}, req)
		if err != nil {
			return nil, err
		}

		httpRes, err = s.sdkConfiguration.Client.Do(req)
		if err != nil || httpRes == nil {
			if err != nil {
				err = fmt.Errorf("error sending request: %w", err)
			} else {
				err = fmt.Errorf("error sending request: no response")
			}
			_, err = s.hooks.AfterError(hooks.AfterErrorContext{HookContext: hookCtx}, nil, err)
			return nil, err
		} else if utils.MatchStatusCodes([]string{"4XX", "5XX"}, httpRes.StatusCode) {
			_httpRes, err := s.hooks.AfterError(hooks.AfterErrorContext{HookContext: hookCtx}, httpRes, nil)
			if err != nil {
				return nil, err
			} else if _httpRes != nil {
				httpRes = _httpRes
			}
		} else {
			httpRes, err = s.hooks.AfterSuccess(hooks.AfterSuccessContext{HookContext: hookCtx}, httpRes)
			if err != nil {
				return nil, err
			}
		}
	}
	return httpRes, nil
}

func (s *DeployServer) prepareOptions(opts []operations.Option) (operations.Options, error) {
	o := operations.Options{}
	supportedOptions := []string{
		operations.SupportedOptionRetries,
		operations.SupportedOptionTimeout,
	}
	for _, opt := range opts {
		if err := opt(&o, supportedOptions...); err != nil {
			return o, fmt.Errorf("error applying option: %w", err)
		}
	}
	return o, nil
}

func (s *DeployServer) baseURL(o operations.Options) string {
	if o.ServerURL == nil {
		return utils.ReplaceParameters(s.sdkConfiguration.GetServerDetails())
	}
	return *o.ServerURL
}

func (s *DeployServer) newHookCtx(ctx context.Context, baseURL, operationID string) hooks.HookContext {
	return hooks.HookContext{
		SDK:              s,
		SDKConfiguration: s.sdkConfiguration,
		BaseURL:          baseURL,
		Context:          ctx,
		OperationID:      operationID,
		OAuth2Scopes:     nil,
		SecuritySource:   nil,
	}
}

func handleErrorResponse(httpRes *http.Response) error {
	rawBody, err := utils.ConsumeRawBody(httpRes)
	if err != nil {
		return err
	}
	return apierrors.NewAPIError("API error occurred", httpRes.StatusCode, string(rawBody), httpRes)
}

// PushManifest - Push a new manifest version
func (s *DeployServer) PushManifest(ctx context.Context, id string, requestBody any, opts ...operations.Option) (*operations.PushManifestResponse, error) {
	request := operations.PushManifestRequest{
		ID:          id,
		RequestBody: requestBody,
	}

	o, err := s.prepareOptions(opts)
	if err != nil {
		return nil, err
	}

	baseURL := s.baseURL(o)
	opURL, err := utils.GenerateURL(ctx, baseURL, "/apps/{id}/manifest", request, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating URL: %w", err)
	}

	hookCtx := s.newHookCtx(ctx, baseURL, "pushManifest")
	bodyReader, reqContentType, err := utils.SerializeRequestBody(ctx, request, false, false, "RequestBody", "raw", `request:"mediaType=application/yaml"`)
	if err != nil {
		return nil, err
	}

	timeout := o.Timeout
	if timeout == nil {
		timeout = s.sdkConfiguration.Timeout
	}
	if timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "POST", opURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", s.sdkConfiguration.UserAgent)
	if reqContentType != "" {
		req.Header.Set("Content-Type", reqContentType)
	}
	for k, v := range o.SetHeaders {
		req.Header.Set(k, v)
	}

	httpRes, err := s.doRequest(ctx, hookCtx, req, o)
	if err != nil {
		return nil, err
	}

	res := &operations.PushManifestResponse{
		HTTPMeta: components.HTTPMetadata{
			Request:  req,
			Response: httpRes,
		},
	}

	switch {
	case httpRes.StatusCode == 201:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.ManifestVersionResponse
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.ManifestVersionResponse = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	case httpRes.StatusCode >= 400 && httpRes.StatusCode < 600:
		return nil, handleErrorResponse(httpRes)
	default:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.Error
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.Error = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	}

	return res, nil
}

// ListManifestVersions - List manifest versions
func (s *DeployServer) ListManifestVersions(ctx context.Context, id string, opts ...operations.Option) (*operations.ListManifestVersionsResponse, error) {
	request := operations.ListManifestVersionsRequest{ID: id}

	o, err := s.prepareOptions(opts)
	if err != nil {
		return nil, err
	}

	baseURL := s.baseURL(o)
	opURL, err := utils.GenerateURL(ctx, baseURL, "/apps/{id}/manifest/versions", request, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating URL: %w", err)
	}

	hookCtx := s.newHookCtx(ctx, baseURL, "listManifestVersions")

	timeout := o.Timeout
	if timeout == nil {
		timeout = s.sdkConfiguration.Timeout
	}
	if timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", opURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", s.sdkConfiguration.UserAgent)
	for k, v := range o.SetHeaders {
		req.Header.Set(k, v)
	}

	httpRes, err := s.doRequest(ctx, hookCtx, req, o)
	if err != nil {
		return nil, err
	}

	res := &operations.ListManifestVersionsResponse{
		HTTPMeta: components.HTTPMetadata{Request: req, Response: httpRes},
	}

	switch {
	case httpRes.StatusCode == 200:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.ListManifestsResponse
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.ListManifestsResponse = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	case httpRes.StatusCode >= 400 && httpRes.StatusCode < 600:
		return nil, handleErrorResponse(httpRes)
	default:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.Error
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.Error = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	}

	return res, nil
}

// CreateDeployment - Create a deployment for an app
func (s *DeployServer) CreateDeployment(ctx context.Context, id string, createDeploymentRequest components.CreateDeploymentRequest, opts ...operations.Option) (*operations.CreateDeploymentResponse, error) {
	request := operations.CreateDeploymentRequest{
		ID:                      id,
		CreateDeploymentRequest: createDeploymentRequest,
	}

	o, err := s.prepareOptions(opts)
	if err != nil {
		return nil, err
	}

	baseURL := s.baseURL(o)
	opURL, err := utils.GenerateURL(ctx, baseURL, "/apps/{id}/deployments", request, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating URL: %w", err)
	}

	hookCtx := s.newHookCtx(ctx, baseURL, "createDeployment")
	bodyReader, reqContentType, err := utils.SerializeRequestBody(ctx, request, false, false, "CreateDeploymentRequest", "json", `request:"mediaType=application/json"`)
	if err != nil {
		return nil, err
	}

	timeout := o.Timeout
	if timeout == nil {
		timeout = s.sdkConfiguration.Timeout
	}
	if timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "POST", opURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", s.sdkConfiguration.UserAgent)
	if reqContentType != "" {
		req.Header.Set("Content-Type", reqContentType)
	}
	for k, v := range o.SetHeaders {
		req.Header.Set(k, v)
	}

	httpRes, err := s.doRequest(ctx, hookCtx, req, o)
	if err != nil {
		return nil, err
	}

	res := &operations.CreateDeploymentResponse{
		HTTPMeta: components.HTTPMetadata{Request: req, Response: httpRes},
	}

	switch {
	case httpRes.StatusCode == 201:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.DeploymentResponse
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.DeploymentResponse = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	case httpRes.StatusCode >= 400 && httpRes.StatusCode < 600:
		return nil, handleErrorResponse(httpRes)
	default:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.Error
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.Error = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	}

	return res, nil
}

// ListDeployments - List deployments for an app
func (s *DeployServer) ListDeployments(ctx context.Context, id string, opts ...operations.Option) (*operations.ListDeploymentsResponse, error) {
	request := operations.ListDeploymentsRequest{ID: id}

	o, err := s.prepareOptions(opts)
	if err != nil {
		return nil, err
	}

	baseURL := s.baseURL(o)
	opURL, err := utils.GenerateURL(ctx, baseURL, "/apps/{id}/deployments", request, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating URL: %w", err)
	}

	hookCtx := s.newHookCtx(ctx, baseURL, "listDeployments")

	timeout := o.Timeout
	if timeout == nil {
		timeout = s.sdkConfiguration.Timeout
	}
	if timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", opURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", s.sdkConfiguration.UserAgent)
	for k, v := range o.SetHeaders {
		req.Header.Set(k, v)
	}

	httpRes, err := s.doRequest(ctx, hookCtx, req, o)
	if err != nil {
		return nil, err
	}

	res := &operations.ListDeploymentsResponse{
		HTTPMeta: components.HTTPMetadata{Request: req, Response: httpRes},
	}

	switch {
	case httpRes.StatusCode == 200:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.ListDeploymentsResponse
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.ListDeploymentsResponse = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	case httpRes.StatusCode >= 400 && httpRes.StatusCode < 600:
		return nil, handleErrorResponse(httpRes)
	default:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.Error
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.Error = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	}

	return res, nil
}

// ReadDeployment - Read a deployment
func (s *DeployServer) ReadDeployment(ctx context.Context, id string, name string, opts ...operations.Option) (*operations.ReadDeploymentResponse, error) {
	request := operations.ReadDeploymentRequest{ID: id, Name: name}

	o, err := s.prepareOptions(opts)
	if err != nil {
		return nil, err
	}

	baseURL := s.baseURL(o)
	opURL, err := utils.GenerateURL(ctx, baseURL, "/apps/{id}/deployments/{name}", request, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating URL: %w", err)
	}

	hookCtx := s.newHookCtx(ctx, baseURL, "readDeployment")

	timeout := o.Timeout
	if timeout == nil {
		timeout = s.sdkConfiguration.Timeout
	}
	if timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", opURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", s.sdkConfiguration.UserAgent)
	for k, v := range o.SetHeaders {
		req.Header.Set(k, v)
	}

	httpRes, err := s.doRequest(ctx, hookCtx, req, o)
	if err != nil {
		return nil, err
	}

	res := &operations.ReadDeploymentResponse{
		HTTPMeta: components.HTTPMetadata{Request: req, Response: httpRes},
	}

	switch {
	case httpRes.StatusCode == 200:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.DeploymentResponse
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.DeploymentResponse = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	case httpRes.StatusCode >= 400 && httpRes.StatusCode < 600:
		return nil, handleErrorResponse(httpRes)
	default:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.Error
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.Error = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	}

	return res, nil
}

// DeleteDeployment - Delete a deployment
func (s *DeployServer) DeleteDeployment(ctx context.Context, id string, name string, opts ...operations.Option) (*operations.DeleteDeploymentResponse, error) {
	request := operations.DeleteDeploymentRequest{ID: id, Name: name}

	o, err := s.prepareOptions(opts)
	if err != nil {
		return nil, err
	}

	baseURL := s.baseURL(o)
	opURL, err := utils.GenerateURL(ctx, baseURL, "/apps/{id}/deployments/{name}", request, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating URL: %w", err)
	}

	hookCtx := s.newHookCtx(ctx, baseURL, "deleteDeployment")

	timeout := o.Timeout
	if timeout == nil {
		timeout = s.sdkConfiguration.Timeout
	}
	if timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", opURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", s.sdkConfiguration.UserAgent)
	for k, v := range o.SetHeaders {
		req.Header.Set(k, v)
	}

	httpRes, err := s.doRequest(ctx, hookCtx, req, o)
	if err != nil {
		return nil, err
	}

	res := &operations.DeleteDeploymentResponse{
		HTTPMeta: components.HTTPMetadata{Request: req, Response: httpRes},
	}

	switch {
	case httpRes.StatusCode == 204:
		utils.DrainBody(httpRes)
	case httpRes.StatusCode >= 400 && httpRes.StatusCode < 600:
		return nil, handleErrorResponse(httpRes)
	default:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.Error
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.Error = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	}

	return res, nil
}

// DeployToDeployment - Deploy a manifest to a deployment
func (s *DeployServer) DeployToDeployment(ctx context.Context, id string, name string, version *int64, opts ...operations.Option) (*operations.DeployToDeploymentResponse, error) {
	request := operations.DeployToDeploymentRequest{
		ID:      id,
		Name:    name,
		Version: version,
	}

	o, err := s.prepareOptions(opts)
	if err != nil {
		return nil, err
	}

	baseURL := s.baseURL(o)
	opURL, err := utils.GenerateURL(ctx, baseURL, "/apps/{id}/deployments/{name}/deploy", request, nil)
	if err != nil {
		return nil, fmt.Errorf("error generating URL: %w", err)
	}

	hookCtx := s.newHookCtx(ctx, baseURL, "deployToDeployment")

	timeout := o.Timeout
	if timeout == nil {
		timeout = s.sdkConfiguration.Timeout
	}
	if timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, "POST", opURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", s.sdkConfiguration.UserAgent)
	if err := utils.PopulateQueryParams(ctx, req, request, nil, nil); err != nil {
		return nil, fmt.Errorf("error populating query params: %w", err)
	}
	for k, v := range o.SetHeaders {
		req.Header.Set(k, v)
	}

	httpRes, err := s.doRequest(ctx, hookCtx, req, o)
	if err != nil {
		return nil, err
	}

	res := &operations.DeployToDeploymentResponse{
		HTTPMeta: components.HTTPMetadata{Request: req, Response: httpRes},
	}

	switch {
	case httpRes.StatusCode == 202:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.RunResponse
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.RunResponse = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	case httpRes.StatusCode >= 400 && httpRes.StatusCode < 600:
		return nil, handleErrorResponse(httpRes)
	default:
		switch {
		case utils.MatchContentType(httpRes.Header.Get("Content-Type"), `application/json`):
			rawBody, err := utils.ConsumeRawBody(httpRes)
			if err != nil {
				return nil, err
			}
			var out components.Error
			if err := utils.UnmarshalJsonFromResponseBody(bytes.NewBuffer(rawBody), &out, ""); err != nil {
				return nil, err
			}
			res.Error = &out
		default:
			return nil, handleErrorResponse(httpRes)
		}
	}

	return res, nil
}
