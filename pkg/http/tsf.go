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
