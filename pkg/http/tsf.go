package http

type Metadata struct {
	ApplicationID      string `json:"ai"`
	ApplicationVersion string `json:"av"`
	ServiceName        string `json:"sn"`
	InstanceID         string `json:"ii"`
	GroupID            string `json:"gi"`
	LocalIP            string `json:"li"`
	NamespaceID        string `json:"ni"`
}

type AtomMetadata struct {
	ApplicationID string `json:"application.id"`
	ServiceName   string `json:"service.name"`
	InstanceID    string `json:"instance.id"`
	GroupID       string `json:"group.id"`
	LocalIP       string `json:"connection.ip"`
	LocalPort     string `json:"service.port"`
	NamespaceID   string `json:"namespace.id"`
	Interface     string `json:"interface"`
}
