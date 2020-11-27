package monitor

import "time"

const (
	CategoryMS = "MS"

	KindClient = "CLIENT"
	KindServer = "SERVER"
)

type Stat struct {
	Begin      time.Time
	End        time.Time
	Category   string
	Kind       string
	Local      *Endpoint
	Remote     *Endpoint
	StatusCode int
}

type Endpoint struct {
	ServiceName   string `json:"service"`
	InterfaceName string `json:"interface"`
	Method        string `json:"method"`
	Path          string `json:"path"`
}

func NewStat(category string, kind string, local *Endpoint, remote *Endpoint) *Stat {
	return &Stat{
		Begin:    time.Now(),
		Category: category,
		Kind:     kind,
		Local:    local,
		Remote:   remote,
	}
}

func (s *Stat) Record(statusCode int) {
	s.End = time.Now()
	s.StatusCode = statusCode
	monitor.saveStat(s)
}

func (s *Stat) HashCode() string {
	hc := s.Category + s.Kind + s.Local.ServiceName + "/" + s.Local.InterfaceName
	if s.Remote == nil {
		return hc
	}
	return hc + "-" + s.Remote.ServiceName + "/" + s.Remote.InterfaceName
}
