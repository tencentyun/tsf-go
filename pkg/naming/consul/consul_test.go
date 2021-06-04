package consul

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/tencentyun/tsf-go/log"
	"github.com/tencentyun/tsf-go/pkg/naming"
	"github.com/tencentyun/tsf-go/pkg/sys/env"
)

var serviceNum int
var nidNum int
var insNum int
var nidStart int
var consulAddr string
var token string
var dereg bool
var appID string
var catalog bool
var subscribe int

func TestMain(m *testing.M) {
	// 这两个参数必传
	flag.StringVar(&consulAddr, "consulAddr", "127.0.0.1:8500", "-consulAddr 127.0.0.1:8500")
	flag.StringVar(&token, "token", "", "-token")

	flag.IntVar(&serviceNum, "serviceNum", 2, "-serviceNum 4")
	flag.IntVar(&nidStart, "nidStart", 0, "-nidStart 0")
	flag.IntVar(&nidNum, "nidNum", 1, "-nidNum 1")
	flag.IntVar(&insNum, "insNum", 10, "-insNum 3")
	flag.StringVar(&appID, "appID", "", "-appID ")
	flag.BoolVar(&catalog, "catalog", true, "-catalog true")
	flag.IntVar(&subscribe, "subscribe", 3, "-subscribe 3")
	flag.BoolVar(&dereg, "dereg", false, "-dereg false")

	flag.Parse()
	m.Run()
}

func TestConsul(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour*4)
	defer cancel()
	fmt.Println("param: ", serviceNum, nidStart, nidNum, insNum, consulAddr, token)
	count := 0
	ch := make(chan struct{}, 0)
	for n := nidStart; n < nidStart+nidNum; n++ {
		for i := 0; i < serviceNum; i++ {
			for j := 0; j < insNum; j++ {
				count++
				go newClient(ctx, ch, fmt.Sprintf("namespace-test-%d", n), "server_test", fmt.Sprintf("server_test_%d_%d", i, j), i)
				time.Sleep(time.Millisecond * 25)
			}
		}
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGHUP)
	sig := <-sigs
	log.DefaultLog.Infow("msg", "[server] got signal,exit now!", "sig", sig.String())
	cancel()
	for i := 0; i < count; i++ {
		<-ch
	}
	time.Sleep(time.Millisecond * 800)
	log.DefaultLog.Info("clear success!")
	return
}

var failCount int64
var successCount int64

func newClient(ctx context.Context, ch chan struct{}, nid string, name string, insID string, idx int) {
	serviceName := fmt.Sprintf("%s-%d", name, idx)
	consul := New(&Config{Address: []string{consulAddr}, Token: token, NamespaceID: nid, AppID: appID, Catalog: catalog})
	ins := naming.Instance{
		ID:      insID + "-" + serviceName,
		Service: &naming.Service{Name: serviceName},
		Host:    env.LocalIP(),
		Port:    8080,
		Metadata: map[string]string{
			"TSF_APPLICATION_ID": "application-maep2nv3",
			"TSF_GROUP_ID":       "group-gyq46ea5",
			"TSF_INSTNACE_ID":    "ins-3jiowz0y",
			"TSF_NAMESPACE_ID":   "namespace-py5lr6v4",
			"TSF_PROG_VERSION":   "provider2",
			"TSF_REGION":         "ap-chongqing",
			"TSF_ZONE":           "",
		},
		Tags: []string{"secure=false"},
	}
	for {
		var err error
		if dereg {
			err = consul.Deregister(&ins)
		} else {
			err = consul.Register(&ins)
		}
		if err != nil {
			failed := atomic.AddInt64(&failCount, 1)
			if failed > atomic.LoadInt64(&successCount) {
				panic(err)
			}
			time.Sleep(time.Second)
		} else {
			atomic.AddInt64(&successCount, 1)
			break
		}
	}
	if dereg {
		ch <- struct{}{}
	}
	time.Sleep(time.Minute * 2)
	for i := 0; i < subscribe; i++ {
		consul.Subscribe(naming.Service{Name: fmt.Sprintf("%s-%d", name, idx+i), Namespace: nid})
	}
	<-ctx.Done()
	s := rand.Int63n(3000)
	time.Sleep(time.Millisecond * time.Duration(s))
	consul.Deregister(&ins)
	ch <- struct{}{}
}
