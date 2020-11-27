package monitor

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/tencentyun/tsf-go/pkg/log"
	"go.uber.org/zap"
)

var monitor *Monitor

func init() {
	monitor = New()
}

func New() *Monitor {
	m := &Monitor{
		current: make(map[string][]*Stat),
	}
	go m.run()
	return m
}

type Monitor struct {
	current map[string][]*Stat
	lock    sync.Mutex
}

func (m *Monitor) saveStat(s *Stat) {
	hc := s.HashCode()
	m.lock.Lock()
	defer m.lock.Unlock()
	m.current[hc] = append(m.current[hc], s)
}

func (m *Monitor) run() {
	ticker := time.NewTicker(time.Second * 60)
	defer ticker.Stop()
	for {
		<-ticker.C
		var old map[string][]*Stat
		m.lock.Lock()
		old = m.current
		m.current = make(map[string][]*Stat)
		m.lock.Unlock()
		go m.dump(old)
	}
}

func (m *Monitor) dump(old map[string][]*Stat) {
	for _, stats := range old {
		if len(stats) == 0 {
			continue
		}
		statusCode := make(map[int]int)
		var rangeMs [11]int64
		var sum float64
		for _, stat := range stats {
			statusCode[stat.StatusCode] = statusCode[stat.StatusCode] + 1
			dur := stat.End.Sub(stat.Begin)
			sum += float64(dur) / float64(time.Millisecond)
			if dur <= time.Millisecond*50 {
				rangeMs[0]++
			} else if dur <= time.Millisecond*100 {
				rangeMs[1]++
			} else if dur <= time.Millisecond*200 {
				rangeMs[2]++
			} else if dur <= time.Millisecond*300 {
				rangeMs[3]++
			} else if dur <= time.Millisecond*400 {
				rangeMs[4]++
			} else if dur <= time.Millisecond*500 {
				rangeMs[5]++
			} else if dur <= time.Millisecond*800 {
				rangeMs[6]++
			} else if dur <= time.Millisecond*1200 {
				rangeMs[7]++
			} else if dur <= time.Millisecond*1600 {
				rangeMs[8]++
			} else if dur <= time.Millisecond*2000 {
				rangeMs[9]++
			} else {
				rangeMs[10]++
			}
		}
		count := len(stats)
		avg := sum / float64(count)
		var statusCodes []StatusCode
		var statusSerial StatusSerial
		for code, num := range statusCode {
			statusCodes = append(statusCodes, StatusCode{Code: strconv.FormatInt(int64(code), 10), Amount: int64(num)})
			if code == 200 {
				statusSerial.Successful++
			} else if code == 400 || code == 499 {
				statusSerial.ClientErr++
			} else if code == 500 {
				statusSerial.ServerErr++
			} else if code == 503 || code == 429 || code == 444 {
				statusSerial.Unavailable++
			} else if code == 504 {
				statusSerial.Timeout++
			} else {
				statusSerial.OtherErr++
			}
		}
		invocation := Invocation{
			SumAmount:  int64(count),
			StatusCode: statusCodes,
			Duration: Duration{
				Avg:     avg,
				Sum:     sum,
				Range50: rangeMs[0],
			},
			StatusSerial: statusSerial,
		}

		var metric MetricItem
		stat := stats[0]
		metric.Kind = stat.Kind
		metric.Cateory = stat.Category
		metric.Timestamp = time.Now().Unix()
		metric.Timestamp = metric.Timestamp - metric.Timestamp%60
		metric.Period = 60
		metric.Local = stat.Local
		metric.Remote = stat.Remote
		metric.Invocation = invocation
		content, err := json.Marshal(metric)
		if err != nil {
			log.L().Error(context.Background(), "Monitor Marshal failed!", zap.Any("metric", metric))
			return
		}
		logger.Info(string(content))
	}
}

type MetricItem struct {
	Cateory    string     `json:"category"`
	Kind       string     `json:"kind"`
	Timestamp  int64      `json:"timestamp"`
	Period     int64      `json:"period"`
	Local      *Endpoint  `json:"local,omitempty"`
	Remote     *Endpoint  `json:"remote,omitempty"`
	Invocation Invocation `json:"invocation"`
}

type Invocation struct {
	SumAmount    int64        `json:"sum_amount"`
	StatusCode   []StatusCode `json:"status_code"`
	StatusSerial StatusSerial `json:"status_serial"`
	Duration     Duration     `json:"duration"`
}

type StatusSerial struct {
	Informational int `json:"informational"`
	Successful    int `json:"successful"`
	Redirection   int `json:"redirection"`
	ClientErr     int `json:"client_error"`
	ServerErr     int `json:"server_error"`
	ConnErr       int `json:"connect_error"`
	Timeout       int `json:"timeout_error"`
	Unavailable   int `json:"unavailable_error"`
	OtherErr      int `json:"other_error"`
}

type StatusCode struct {
	Code   string `json:"code"`
	Amount int64  `json:"amount"`
}

type Duration struct {
	Avg       float64 `json:"avg_ms"`
	Sum       float64 `json:"sum_ms"`
	Range50   int64   `json:"range_50_ms,omitempty"`
	Range100  int64   `json:"range_50_100_ms,omitempty"`
	Range200  int64   `json:"range_100_200_ms,omitempty"`
	Range300  int64   `json:"range_200_300_ms,omitempty"`
	Range400  int64   `json:"range_300_400_ms,omitempty"`
	Range500  int64   `json:"range_400_500_ms,omitempty"`
	Range800  int64   `json:"range_500_800_ms,omitempty"`
	Range1200 int64   `json:"range_800_1200_ms,omitempty"`
	Range1600 int64   `json:"range_1200_1600_ms,omitempty"`
	Range2000 int64   `json:"range_1600_2000_ms,omitempty"`
	RangeInf  int64   `json:"range_2000_ms,omitempty"`
}
