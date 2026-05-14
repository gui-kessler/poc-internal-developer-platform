package spec

type Intent struct {
	App          string   `json:"app"`
	Image        string   `json:"image"`
	Replicas     int      `json:"replicas"`
	Port         int      `json:"port"`
	Dependencies []string `json:"dependencies,omitempty"`
}
