package command

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/formancehq/fctl/pkg/pluginsdk"
)

func StringFlag(name, usage string) pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: name, Type: "string", Usage: usage}
}
func BoolFlag(name, usage string) pluginsdk.FlagSpec {
	return pluginsdk.FlagSpec{Name: name, Type: "bool", Default: "false", Usage: usage}
}
func PaginationFlags() []pluginsdk.FlagSpec {
	return []pluginsdk.FlagSpec{StringFlag("cursor", "Opaque pagination cursor"), {Name: "page-size", Type: "uint32", Default: "100", Usage: "Page size (1-1000)"}}
}
func DataFlag() pluginsdk.FlagSpec {
	f := StringFlag("data", "JSON payload, @file or - (read by the host)")
	f.Body = true
	return f
}

func Leaf(command, use, method, path string, minArgs, maxArgs int, flags ...pluginsdk.FlagSpec) Operation {
	op := Operation{Command: command, Method: method, Segments: strings.Split(strings.Trim(path, "/"), "/"),
		Spec: pluginsdk.CommandSpec{Use: use, Short: strings.ToUpper(command[:1]) + command[1:], Runnable: true, Args: pluginsdk.ArgsSpec{Min: minArgs, Max: maxArgs}, Flags: flags}}
	for i := range minArgs {
		op.Spec.Inputs = append(op.Spec.Inputs, pluginsdk.InputSpec{Title: fmt.Sprintf("%s argument %d", command, i+1), Kind: "text", Argument: new(i), Required: true})
	}
	return op
}

func Confirmed(op Operation) Operation {
	op.Spec.Confirm = true
	op.Spec.Flags = append(op.Spec.Flags, BoolFlag("confirm", "Confirm this operation"))
	return op
}

func Payload(op Operation, fields ...string) Operation {
	// Historical file/stdin/URL input stays a host-owned boundary. Direct
	// callers and forms can still supply Body without the optional source.
	source := op.Spec.Args.Max
	op.Spec.Files = &pluginsdk.FileSpec{ReadArgument: &source, ReadFormat: "yaml"}
	op.Spec.Args.Max++
	op.Spec.Use += " [<file>|-]"
	op.Spec.Flags = append(op.Spec.Flags, DataFlag())
	op.Spec.Inputs = append(op.Spec.Inputs, pluginsdk.InputSpec{Title: "JSON payload", Kind: "text", ValueType: "string", Flag: "data", Required: true})
	op.Body = func(r pluginsdk.ExecuteRequest) (json.RawMessage, error) { return ObjectBody(r.Body, fields...) }
	return op
}
