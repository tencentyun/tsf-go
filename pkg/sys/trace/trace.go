package trace

import (
	"sync"

	"github.com/openzipkin/zipkin-go/reporter"
)

var report reporter.Reporter
var mu sync.Mutex

func GetReporter() reporter.Reporter {
	mu.Lock()
	defer mu.Unlock()
	if report == nil {
		report = &tsfReporter{logger: DefaultLogger}
	}
	return report
}

func CloseReporter() {
	report.Close()
}
