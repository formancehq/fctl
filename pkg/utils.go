package fctl

import (
	"encoding/json"
	"errors"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
	"path"
	"runtime"
)

func StructToMap(obj interface{}) (newMap map[string]interface{}, err error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &newMap)
	return
}

func Map[SRC any, DST any](srcs []SRC, mapper func(SRC) DST) []DST {
	ret := make([]DST, 0)
	for _, src := range srcs {
		ret = append(ret, mapper(src))
	}
	return ret
}

func MapMap[KEY comparable, VALUE any, DST any](srcs map[KEY]VALUE, mapper func(KEY, VALUE) DST) []DST {
	ret := make([]DST, 0)
	for k, v := range srcs {
		ret = append(ret, mapper(k, v))
	}
	return ret
}

func Prepend[V any](array []V, items ...V) []V {
	return append(items, array...)
}

func ContainValue[V comparable](array []V, value V) bool {
	for _, v := range array {
		if v == value {
			return true
		}
	}
	return false
}

var (
	ErrOpeningBrowser = errors.New("opening browser")
)

func OpenURL(url string) error {
	var (
		cmd  string
		args []string
	)

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	_, err := exec.LookPath(cmd)
	if err == nil {
		args = append(args, url)
		return exec.Command(cmd, args...).Start()
	}

	return ErrOpeningBrowser
}

func ReadJSONFile[V any](cmd *cobra.Command, filePath string) (*V, error) {
	f, err := os.Open(GetFilePath(cmd, filePath))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	v := new(V)
	if err := json.NewDecoder(f).Decode(v); err != nil {
		return nil, err
	}

	return v, nil
}

func WriteJSONFile(filePath string, data any) error {
	dir := path.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return err
	}
	return nil
}
