package payments

import (
	"net/http"

	commandapi "github.com/formancehq/fctl/misc/fctl-plugin/internal/command"
)

func taskOperations() []commandapi.Operation {
	return []commandapi.Operation{
		commandapi.Leaf("tasks get", "get <task-id>", http.MethodGet, "tasks/$0", 1, 1),
	}
}
