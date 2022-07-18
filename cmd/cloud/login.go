package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

type AuthenticateRequest struct {
	Email string `json:"email"`
}

func Authenticate(email string) (string, error) {
	request := &AuthenticateRequest{
		Email: email,
	}

	b, _ := json.Marshal(request)

	res, err := http.Post(
		fmt.Sprintf("%s/auth/authenticate", CloudURI),
		"application/json",
		bytes.NewBuffer(b),
	)

	r, _ := io.ReadAll(res.Body)

	return string(r), err
}

func NewLogin() *cobra.Command {
	return &cobra.Command{
		Use:  "login [email]",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			Authenticate(args[0])
		},
	}
}
