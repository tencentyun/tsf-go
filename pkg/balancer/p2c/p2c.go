package p2c

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tencentyun/tsf-go/pkg/balancer"
	"github.com/tencentyun/tsf-go/pkg/log"
	"github.com/tencentyun/tsf-go/pkg/naming"
)

var _ balancer.Balancer = &P2cPicker{}

const (
	// The mean lifetime of `cost`, it reaches its half-life after Tau*ln(2).
	tau = int64(time.Millisecond * 800)
	// if statistic not collected,we add a big lag penalty to endpoint
	penalty = uint64(time.Second * 20)

	forceGap = int64(time.Second * 3)

	Name = "p2c"
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
}

func newSubConn(node *naming.Instance) *subConn {
	return &subConn{
		node:     node,
		lag:      0,
		success:  1000,
		inflight: 1,
	}
}

func (sc *subConn) valid() bool {
	return sc.health() >= 500
}

func (sc *subConn) health() uint64 {
	return atomic.LoadUint64(&sc.success)
}

func (sc *subConn) load() uint64 {
	lag := uint64(math.Sqrt(float64(atomic.LoadUint64(&sc.lag))) + 1)
	load := lag * uint64(atomic.LoadInt64(&sc.inflight))
	if load == 0 {
		// penalty是初始化没有数据时的惩罚值，默认为1e9 * 20
		load = penalty * uint64(atomic.LoadInt64(&sc.inflight))
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

// Builder is p2c Builder
type Builder struct{}

// Build p2c
func (*Builder) Build(ctx context.Context, nodes []naming.Instance, errHandler func(error) bool) balancer.Balancer {
	p := &P2cPicker{
		r:          rand.New(rand.NewSource(time.Now().UnixNano())),
		subConns:   make(map[string]*subConn),
		errHandler: errHandler,
	}
	for i := range nodes {
		p.subConns[nodes[i].Addr()] = newSubConn(&nodes[i])
	}

	return p
}

type P2cPicker struct {
	// subConns is the snapshot of the weighted-roundrobin balancer when this picker was
	// created. The slice is immutable. Each Get() will do a round robin
	// selection from it and return the selected SubConn.
	subConns   map[string]*subConn
	logTs      int64
	r          *rand.Rand
	lk         sync.Mutex
	errHandler func(err error) (isErr bool)
}

// choose two distinct nodes
func (p *P2cPicker) prePick(nodes []naming.Instance) (nodeA *subConn, nodeB *subConn) {
	for i := 0; i < 2; i++ {
		p.lk.Lock()
		a := p.r.Intn(len(nodes))
		b := p.r.Intn(len(nodes) - 1)
		if b >= a {
			b = b + 1
		}
		nodeA, nodeB = p.subConns[nodes[a].Addr()], p.subConns[nodes[b].Addr()]
		if nodeA == nil {
			nodeA = newSubConn(&nodes[a])
			p.subConns[nodeA.node.Addr()] = nodeA
		}
		if nodeB == nil {
			nodeB = newSubConn(&nodes[b])
			p.subConns[nodeA.node.Addr()] = nodeB
		}
		p.lk.Unlock()

		if nodeA.valid() || nodeB.valid() {
			break
		}
	}
	return
}

func (p *P2cPicker) Pick(ctx context.Context, nodes []naming.Instance) (*naming.Instance, func(di balancer.DoneInfo)) {
	var pc, upc *subConn
	start := time.Now().UnixNano()

	if len(nodes) == 0 {
		return nil, func(di balancer.DoneInfo) {}
	} else if len(nodes) == 1 {
		p.lk.Lock()
		pc = p.subConns[nodes[0].Addr()]
		if pc == nil {
			pc = newSubConn(&nodes[0])
			p.subConns[nodes[0].Addr()] = pc
		}
		p.lk.Unlock()
	} else {
		nodeA, nodeB := p.prePick(nodes)
		// meta.Weight为服务发布者在disocvery中设置的权重
		if nodeA.load()*nodeB.health() > nodeB.load()*nodeA.health() {
			pc, upc = nodeB, nodeA
		} else {
			pc, upc = nodeA, nodeB
		}
		// 如果选中的节点，在forceGap期间内没有被选中一次，那么强制一次
		// 利用强制的机会，来触发成功率、延迟的衰减
		// 原子锁conn.pick保证并发安全，放行一次
		pick := atomic.LoadInt64(&upc.pick)
		if start-pick > forceGap && atomic.CompareAndSwapInt64(&upc.pick, pick, start) {
			pc = upc
		}
	}

	// 节点未发生切换才更新pick时间
	if pc != upc {
		atomic.StoreInt64(&pc.pick, start)
	}
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

		logTs := atomic.LoadInt64(&p.logTs)
		if now-logTs > int64(time.Second*3) {
			if atomic.CompareAndSwapInt64(&p.logTs, logTs, now) {
				p.PrintStats()
			}
		}
	}
}

func (p *P2cPicker) PrintStats() {
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
		log.Debugf(context.Background(), "p2c %s : %+v", serverName, stats)
	}
}

func (p *P2cPicker) Schema() string {
	return Name
}
