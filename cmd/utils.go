package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func metadataFromFlag(flag string) (map[string]any, error) {
	metadata := map[string]interface{}{}
	for _, v := range viper.GetStringSlice(flag) {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 1 {
			return nil, fmt.Errorf("malformed metadata: %s", v)
		}
		metadata[parts[0]] = parts[1]
	}
	return metadata, nil
}
