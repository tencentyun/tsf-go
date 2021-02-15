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
	"github.com/tencentyun/tsf-go/pkg/naming"
)

var serverNum int
var cliNum int
var concurrency int
var extraLoad int64
var extraDelay int64
var extraWeight uint64
var chaos int

func init() {
	flag.IntVar(&serverNum, "snum", 6, "-snum 6")
	flag.IntVar(&cliNum, "cnum", 12, "-cnum 12")
	flag.IntVar(&concurrency, "concurrency", 10, "-cc 10")
	flag.Int64Var(&extraLoad, "exload", 3, "-exload 3")
	flag.Int64Var(&extraDelay, "exdelay", 250, "-exdelay 250")
	flag.IntVar(&chaos, "chaos", 2, "-chaos 2")

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
		wait: make(chan struct{}, 1000),
	}
	go func() {
		for {
			for i := 0; i < 100; i++ {
				<-sc.wait
			}
			time.Sleep(time.Millisecond * 10)
		}
	}()

	return
}

func (s *testSubConn) connect(ctx context.Context) {
	start := time.Now()
	time.Sleep(time.Millisecond * 10)
	//add qps counter when request come in
	select {
	case <-ctx.Done():
		return
	case s.wait <- struct{}{}:
	}
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
	atomic.AddInt64(&s.reqs, 1)
	atomic.AddUint64(&s.lag, uint64(time.Since(start).Milliseconds()))
}

func TestChaosPick(t *testing.T) {
	flag.Parse()
	t.Logf("start chaos test!svrNum:%d cliNum:%d concurrency:%d exLoad:%d exDelay:%d\n", serverNum, cliNum, concurrency, extraLoad, extraDelay)
	c := newController(serverNum, cliNum)
	c.launch(concurrency)
	go c.control(extraLoad, extraDelay)
	time.Sleep(time.Second * 50)
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

	for i := 0; i < cliNum; i++ {
		b := p2cBuilder.Build(context.Background(), nodes, nil)
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
					time.Sleep(time.Millisecond * 10)
				}
			}()
		}
	}
}

func (c *controller) control(extraLoad, extraDelay int64) {
	for {
		fmt.Printf("\n")
		//make some chaos
		n := rand.Intn(chaos + 1)
		for i := 0; i < n; i++ {
			if extraLoad > 0 {
				degree := rand.Int63n(extraLoad)
				degree++
				atomic.StoreInt64(&c.serverSet[i].loadJitter, degree)
				fmt.Printf("set addr_%d load:%d ", i, degree)
			}
			if extraDelay > 0 {
				degree := rand.Int63n(extraDelay)
				atomic.StoreInt64(&c.serverSet[i].delayJitter, degree)
				fmt.Printf("set addr_%d delay:%dms ", i, degree)
			}
		}

		for i := range c.serverSet {
			sc := c.serverSet[i]
			atomic.StoreInt64(&sc.reqs, 0)
			atomic.StoreUint64(&sc.lag, 0)
		}
		sleep := int64(10)
		time.Sleep(time.Second * 10)
		fmt.Printf("\n")
		for _, sc := range c.servers {
			req := atomic.SwapInt64(&sc.reqs, 0)
			lag := atomic.SwapUint64(&sc.lag, 0)
			lagAvg := float64(lag) / float64(req)
			qps := req / sleep
			wait := len(sc.wait)
			fmt.Printf("%s qps:%d lag:%v waits:%d\n", sc.node.Addr(), qps, lagAvg, wait)
		}
		/*for _, picker := range c.clients {
			p := picker.(*p2c.P2cPicker)
			p.PrintStats()
		}*/
		fmt.Printf("\n")
		//reset chaos
		for i := range c.serverSet {
			atomic.StoreInt64(&c.serverSet[i].loadJitter, 0)
			atomic.StoreInt64(&c.serverSet[i].delayJitter, 0)
		}
		time.Sleep(time.Second * 3)
	}
}
