package ledger

import (
	"context"
	"fmt"

	formance "github.com/formancehq/formance-sdk-go/v3"
	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/fctl/v4/internal/capabilities"
)

const FeatureGetInfo capabilities.Feature = "getInfo"

type ReadInfoOutput struct {
	APIVersion capabilities.APIVersion `json:"apiVersion" yaml:"apiVersion"`
	Server     string                  `json:"server" yaml:"server"`
	Version    string                  `json:"version" yaml:"version"`
}

type ReadInfoHandler struct {
	APIVersion capabilities.APIVersion
	Run        func(context.Context) (ReadInfoOutput, error)
}

type ReadInfoService struct {
	Handlers []ReadInfoHandler
	Resolve  func(context.Context, []capabilities.APIVersion) (capabilities.APIVersion, error)
}

func (s ReadInfoService) Run(ctx context.Context) (ReadInfoOutput, error) {
	handlerVersions := make([]capabilities.APIVersion, 0, len(s.Handlers))
	handlers := map[capabilities.APIVersion]ReadInfoHandler{}
	for _, handler := range s.Handlers {
		handlerVersions = append(handlerVersions, handler.APIVersion)
		handlers[handler.APIVersion] = handler
	}

	selected, err := s.Resolve(ctx, handlerVersions)
	if err != nil {
		return ReadInfoOutput{}, err
	}
	handler, ok := handlers[selected]
	if !ok {
		return ReadInfoOutput{}, fmt.Errorf("resolved api version %s has no handler", selected)
	}

	output, err := handler.Run(ctx)
	if err != nil {
		return ReadInfoOutput{}, err
	}
	output.APIVersion = selected
	return output, nil
}

func SDKReadInfoHandlers(sdk *formance.Formance) []ReadInfoHandler {
	return []ReadInfoHandler{
		{
			APIVersion: "v1",
			Run: func(ctx context.Context) (ReadInfoOutput, error) {
				response, err := sdk.Ledger.V1.GetInfo(ctx)
				if err != nil {
					return ReadInfoOutput{}, err
				}
				if response.ConfigInfoResponse == nil {
					return ReadInfoOutput{}, fmt.Errorf("ledger v1 info returned no data")
				}
				return fromV1Info(response.ConfigInfoResponse.Data), nil
			},
		},
	}
}

func fromV1Info(info shared.ConfigInfo) ReadInfoOutput {
	return ReadInfoOutput{
		Server:  info.Server,
		Version: info.Version,
	}
}
