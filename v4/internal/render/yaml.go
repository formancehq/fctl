package render

import (
	"io"

	"gopkg.in/yaml.v3"
)

func YAML(w io.Writer, value any) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	defer encoder.Close()
	return encoder.Encode(value)
}
