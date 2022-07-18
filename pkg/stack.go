package fctl

type Stack struct {
	Version  string     `json:"version"`
	Services []*Service `json:"services"`
}

type Service struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

func NewStack() *Stack {
	stack := Stack{
		Services: []*Service{},
	}

	stack.Services = append(
		stack.Services,
		&Service{
			Name: "ledger",
		},
		&Service{
			Name: "payments",
		},
	)

	return &stack
}
