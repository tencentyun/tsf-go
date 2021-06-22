package wrr

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/pkg/balancer"
	"github.com/tencentyun/tsf-go/pkg/naming"
)

var (
	_ balancer.Balancer  = &WrrPicker{}
	_ balancer.Printable = &WrrPicker{}
)

const (
	// The mean lifetime of `cost`, it reaches its half-life after Tau*ln(2).
	tau = int64(time.Millisecond * 100)
	// if statistic not collected,we add a big lag penalty to endpoint
	penalty = uint64(time.Second * 20)

	updateGap = time.Millisecond * 1600

	Name = "wrr"
)

// Name is the name of pick of two random choices balancer.

// NewBuilder creates a new weighted-roundrobin balancer builder.
func NewBuilder() *Builder {
	return &Builder{}
}

type subConn struct {
	// node
	node *naming.Instance
	//client statistic data
	lag      uint64
	success  uint64
	inflight int64

	//last collected timestamp
	stamp int64
	//last pick timestamp
	pick int64
	// request number in a period time
	reqs int64

	score float64

	// current weight
	cwt float64
}

func newSubConn(node *naming.Instance) *subConn {
	return &subConn{
		node:     node,
		lag:      0,
		success:  1000,
		inflight: 1,
	}
}

func (sc *subConn) health() uint64 {
	return atomic.LoadUint64(&sc.success)
}

func (sc *subConn) EWT() float64 {
	if sc.score == 0 {
		return 100 / float64(sc.inFlight())
	}
	return sc.score / float64(sc.inFlight())
}

func (sc *subConn) inFlight() int64 {
	return atomic.LoadInt64(&sc.inflight)
}

func (sc *subConn) load() uint64 {
	load := uint64(atomic.LoadUint64(&sc.lag) + 1)
	if load == 0 {
		// penalty是初始化没有数据时的惩罚值，默认为1e9 * 20
		load = penalty
	}
	return load
}

// statistics is info for log
type statistic struct {
	addr     string
	score    float64
	cs       uint64
	lantency uint64
	inflight int64
	reqs     int64
}

// Builder is wrr Builder
type Builder struct{}

// Build wrr
func (*Builder) Build(ctx context.Context, nodes []naming.Instance, errHandler func(error) bool, withoutFlight bool, withoutLag bool) balancer.Balancer {
	p := &WrrPicker{
		r:             rand.New(rand.NewSource(time.Now().UnixNano())),
		subConns:      make(map[string]*subConn),
		errHandler:    errHandler,
		updateAt:      time.Now().UnixNano(),
		withoutFlight: withoutFlight,
		withoutLag:    withoutLag,
	}
	for i := range nodes {
		p.subConns[nodes[i].Addr()] = newSubConn(&nodes[i])
	}

	return p
}

type WrrPicker struct {
	// subConns is the snapshot of the weighted-roundrobin balancer when this picker was
	// created. The slice is immutable. Each Get() will do a round robin
	// selection from it and return the selected SubConn.
	subConns   map[string]*subConn
	logTs      int64
	r          *rand.Rand
	lk         sync.Mutex
	errHandler func(err error) (isErr bool)

	updateAt      int64
	withoutFlight bool
	withoutLag    bool
}

func (p *WrrPicker) Pick(ctx context.Context, nodes []naming.Instance) (*naming.Instance, func(di balancer.DoneInfo)) {
	var (
		pc          *subConn
		totalWeight float64
	)

	if len(nodes) == 0 {
		return nil, func(di balancer.DoneInfo) {}
	}
	start := time.Now().UnixNano()

	p.lk.Lock()
	// nginx wrr load balancing algorithm: http://blog.csdn.net/zhangskd/article/details/50194069
	for _, sc := range p.subConns {
		ewt := sc.EWT()
		totalWeight += ewt
		sc.cwt += ewt
		if pc == nil || pc.cwt < sc.cwt {
			pc = sc
		}
	}
	pc.cwt -= totalWeight
	p.lk.Unlock()
	atomic.AddInt64(&pc.inflight, 1)
	atomic.AddInt64(&pc.reqs, 1)

	return pc.node, func(di balancer.DoneInfo) {
		atomic.AddInt64(&pc.inflight, -1)
		now := time.Now().UnixNano()
		// get moving average ratio w
		stamp := atomic.SwapInt64(&pc.stamp, now)
		td := now - stamp
		if td < 0 {
			td = 0
		}
		w := math.Exp(float64(-td) / float64(tau))

		lag := now - start
		if lag < 0 {
			lag = 0
		}
		oldLag := atomic.LoadUint64(&pc.lag)
		if oldLag == 0 {
			w = 0.0
		}
		lag = int64(float64(oldLag)*w + float64(lag)*(1.0-w))
		atomic.StoreUint64(&pc.lag, uint64(lag))

		success := uint64(1000) // error value ,if error set 1
		if di.Err != nil {
			if errors.Is(context.DeadlineExceeded, di.Err) || errors.Is(context.Canceled, di.Err) {
				success = 0
			} else if p.errHandler != nil && p.errHandler(di.Err) {
				success = 0
			}
		}
		oldSuc := atomic.LoadUint64(&pc.success)
		success = uint64(float64(oldSuc)*w + float64(success)*(1.0-w))
		atomic.StoreUint64(&pc.success, success)

		u := atomic.LoadInt64(&p.updateAt)
		if now-u < int64(updateGap) {
			return
		}
		if !atomic.CompareAndSwapInt64(&p.updateAt, u, now) {
			return
		}
		var (
			count int
			total float64
		)
		for _, conn := range p.subConns {
			if p.withoutLag {
				conn.score = float64(conn.health() * 1e7)
			} else {
				conn.score = float64(conn.health()*1e7) / float64(conn.load())

			}

			if conn.score > 0 {
				total += conn.score
				count++
			}
		}
		// count must be greater than 1,otherwise will lead ewt to 0
		if count < 2 {
			return
		}
		avgscore := total / float64(count)
		p.lk.Lock()
		for _, conn := range p.subConns {
			if conn.score <= 0 {
				conn.score = avgscore / 4
			}
		}
		p.lk.Unlock()
	}
}

func (p *WrrPicker) PrintStats() {
	if len(p.subConns) == 0 {
		return
	}
	stats := make([]statistic, 0, len(p.subConns))
	var serverName string
	var reqs int64
	for _, conn := range p.subConns {
		var stat statistic
		stat.addr = conn.node.Addr()
		stat.cs = atomic.LoadUint64(&conn.success)
		stat.inflight = atomic.LoadInt64(&conn.inflight)
		stat.lantency = atomic.LoadUint64(&conn.lag)
		stat.reqs = atomic.SwapInt64(&conn.reqs, 0)
		load := conn.load()
		if load != 0 {
			stat.score = float64(stat.cs*1e8) / float64(load)
		}
		stats = append(stats, stat)
		if serverName == "" {
			serverName = conn.node.Service.Name
		}
		reqs += stat.reqs
	}
	if reqs > 10 {
		log.DefaultLog.Debugf("p2c %s : %+v", serverName, stats)
	}
}

func (p *WrrPicker) Schema() string {
	return Name
}
