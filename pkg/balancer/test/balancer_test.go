package test

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tencentyun/tsf-go/pkg/balancer"
	"github.com/tencentyun/tsf-go/pkg/balancer/p2c"
	"github.com/tencentyun/tsf-go/pkg/balancer/random"
	"github.com/tencentyun/tsf-go/pkg/balancer/wrr"
	"github.com/tencentyun/tsf-go/pkg/naming"
)

var serverNum int
var cliNum int
var concurrency int
var extraLoad int64
var extraDelay int64
var extraWeight uint64
var chaos int
var picker string

func init() {
	flag.IntVar(&serverNum, "snum", 3, "-snum 6")
	flag.IntVar(&cliNum, "cnum", 6, "-cnum 12")
	flag.IntVar(&concurrency, "concurrency", 18, "-cc 10")
	flag.Int64Var(&extraLoad, "exload", 2, "-exload 3")
	flag.Int64Var(&extraDelay, "exdelay", 80, "-exdelay 250")
	flag.IntVar(&chaos, "chaos", 1, "-chaos 1")
	flag.StringVar(&picker, "picker", "wrr", "-picker p2c")
}

type testSubConn struct {
	node naming.Instance
	wait chan struct{}
	//statics
	reqs int64
	lag  uint64
	//control params
	loadJitter  int64
	delayJitter int64
}

func newTestSubConn(addr string) (sc *testSubConn) {
	sc = &testSubConn{
		node: naming.Instance{Host: addr, Port: 8080, Service: &naming.Service{Name: "test-svr"}},
		wait: make(chan struct{}, 120),
	}
	for i := 0; i < 20; i++ {
		go func() {
			for {
				<-sc.wait
				if len(sc.wait) > 110 {
					time.Sleep(time.Millisecond * 10)
				} else if len(sc.wait) > 90 {
					time.Sleep(time.Millisecond * 5)
				}
				time.Sleep(time.Millisecond * 10)
			}
		}()
	}
	return
}

func (s *testSubConn) connect(ctx context.Context) {
	start := time.Now()
	time.Sleep(time.Millisecond * 100)
	//add qps counter when request come in
	select {
	case <-ctx.Done():
		return
	case s.wait <- struct{}{}:
	}
	should := rand.Intn(100)
	if should < 9 {
		load := atomic.LoadInt64(&s.loadJitter)
		if load > 0 {
			for i := 0; i <= rand.Intn(int(load)); i++ {
				select {
				case <-ctx.Done():
					return
				case s.wait <- struct{}{}:
				}
			}
		}
		delay := atomic.LoadInt64(&s.delayJitter)
		if delay > 0 {
			delay = rand.Int63n(delay)
			time.Sleep(time.Millisecond * time.Duration(delay))
		}
	}

	atomic.AddInt64(&s.reqs, 1)
	atomic.AddUint64(&s.lag, uint64(time.Since(start).Milliseconds()))
}

func TestChaosPick(t *testing.T) {
	flag.Parse()
	t.Logf("start chaos test!pciker:%s svrNum:%d cliNum:%d chaos:%d concurrency:%d exLoad:%d exDelay:%d\n", picker, serverNum, cliNum, chaos, concurrency, extraLoad, extraDelay)
	c := newController(serverNum, cliNum)
	c.launch(concurrency)
	c.control(extraLoad, extraDelay)
}

func newController(svrNum int, cliNum int) *controller {
	//new servers
	servers := map[string]*testSubConn{}
	serverSet := []*testSubConn{}
	var nodes []naming.Instance

	var weight uint64 = 10
	if extraWeight > 0 {
		weight = extraWeight
	}
	for i := 0; i < svrNum; i++ {
		weight += extraWeight
		sc := newTestSubConn(fmt.Sprintf("addr_%d", i))
		nodes = append(nodes, sc.node)
		servers[sc.node.Addr()] = sc
		serverSet = append(serverSet, sc)
	}
	//new clients
	var clients []balancer.Balancer

	p2cBuilder := p2c.NewBuilder()
	wrrBuilder := wrr.NewBuilder()

	for i := 0; i < cliNum; i++ {
		var b balancer.Balancer
		if picker == "p2c" {
			b = p2cBuilder.Build(context.Background(), nodes, nil)
		} else if picker == "random" {
			b = &random.Picker{}
		} else if picker == "wrr" {
			b = wrrBuilder.Build(context.Background(), nodes, nil)
		}
		clients = append(clients, b)
	}

	c := &controller{
		serverSet: serverSet,
		servers:   servers,
		clients:   clients,
		nodes:     nodes,
	}
	return c
}

type controller struct {
	servers   map[string]*testSubConn
	serverSet []*testSubConn
	clients   []balancer.Balancer
	nodes     []naming.Instance
}

func (c *controller) launch(concurrency int) {
	bkg := context.Background()
	for i := range c.clients {
		for j := 0; j < concurrency; j++ {
			picker := c.clients[i]
			go func() {
				for {
					go func() {
						ctx, cancel := context.WithTimeout(bkg, time.Millisecond*1000)
						sc, done := picker.Pick(ctx, c.nodes)
						server := c.servers[sc.Addr()]
						server.connect(ctx)
						var err error
						if ctx.Err() != nil {
							err = ctx.Err()
						}
						cancel()
						done(balancer.DoneInfo{Err: err})
					}()
					time.Sleep(time.Millisecond * 20)
				}
			}()
		}
	}
}

func (c *controller) control(extraLoad, extraDelay int64) {
	start := time.Now()
	for j := 0; j < 20; j++ {
		fmt.Printf("\n")
		//make some chaos
		n := chaos
		for i := 0; i < n; i++ {
			chosen := rand.Intn(len(c.serverSet))
			if extraLoad > 0 {
				degree := extraLoad
				atomic.StoreInt64(&c.serverSet[chosen].loadJitter, degree)
				fmt.Printf("set addr_%d load:%d ", chosen, degree)
			}
			if extraDelay > 0 {
				degree := extraDelay
				atomic.StoreInt64(&c.serverSet[chosen].delayJitter, degree)
				fmt.Printf("set addr_%d delay:%dms ", chosen, degree)
			}
		}
		fmt.Printf("\n")

		time.Sleep(time.Millisecond * 1000)

		for _, picker := range c.clients {
			p, ok := picker.(balancer.Printable)
			if ok {
				p.PrintStats()
			}
		}
		//reset chaos
		for i := range c.serverSet {
			atomic.StoreInt64(&c.serverSet[i].loadJitter, 0)
			atomic.StoreInt64(&c.serverSet[i].delayJitter, 0)
		}
		//time.Sleep(time.Second * 3)
	}
	gap := time.Since(start)
	fmt.Printf("\n")

	for _, sc := range c.servers {
		req := atomic.LoadInt64(&sc.reqs)
		lag := atomic.LoadUint64(&sc.lag)
		lagAvg := float64(lag) / float64(req)
		qps := float64(req) / gap.Seconds()
		wait := len(sc.wait)
		fmt.Printf("%s qps:%v lag:%v waits:%d\n", sc.node.Addr(), qps, lagAvg, wait)
	}
}
