package processes

type ProcessState struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type Manager struct {
	Children []ProcessState `json:"children"`
}

func NewManager() Manager {
	return Manager{
		Children: []ProcessState{},
	}
}
