package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"

	pb "github.com/tencentyun/tsf-go/testdata"

	"google.golang.org/grpc"
)

var success int64
var failure int64
var max int64

var n int
var isGrpc bool
var d time.Duration

func init() {
	flag.IntVar(&n, "n", 10, "-n 10")
	flag.BoolVar(&isGrpc, "grpc", true, "-grpc true")
	flag.DurationVar(&d, "d", time.Second*30, "-d 30s")
}

func main() {
	flag.Parse()
	fmt.Println(n, isGrpc, d)
	if isGrpc {
		go grpcBench()
	} else {
		go httpBench()
	}
	time.Sleep(d)

	total := atomic.LoadInt64(&success) + atomic.LoadInt64(&failure)
	qps := float64(total) / d.Seconds()
	fmt.Printf("qps: %v mean: %v max: %v\n ", qps, time.Duration(int64(d)/total), time.Duration(atomic.LoadInt64(&max)))
	fmt.Println("success: ", success, "failure: ", failure)
}

func grpcBench() {
	for i := 0; i < n; i++ {
		go func() {
			cc, err := grpc.Dial("127.0.0.1:8080", grpc.WithInsecure())
			if err != nil {
				panic(err)
			}
			greeter := pb.NewGreeterClient(cc)
			for {
				start := time.Now()
				err = grpcCall(greeter)
				temp := time.Since(start)
				if int64(temp) > atomic.LoadInt64(&max) {
					atomic.StoreInt64(&max, int64(temp))
				}
				if err != nil {
					atomic.AddInt64(&failure, 1)
				} else {
					atomic.AddInt64(&success, 1)
				}
			}
		}()
	}
}

func httpBench() {
	for i := 0; i < n; i++ {
		go func() {
			client := &http.Client{Timeout: time.Second * 3}
			for {
				start := time.Now()

				err := httpCall(client)
				temp := time.Since(start)
				if int64(temp) > atomic.LoadInt64(&max) {
					atomic.StoreInt64(&max, int64(temp))
				}
				if err != nil {
					atomic.AddInt64(&failure, 1)
				} else {
					atomic.AddInt64(&success, 1)
				}
			}
		}()

	}
}

func grpcCall(client pb.GreeterClient) error {
	in := &pb.HelloRequest{Name: "longxia"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	reply, err := client.SayHello(ctx, in)
	if err != nil {
		return err
	}
	if reply.Message == "" {
		return fmt.Errorf("invalid resp")
	}
	return nil
}

func httpCall(client *http.Client) error {
	client = &http.Client{Timeout: time.Second * 3}

	content := `{"name":"longxia"}`
	req, err := http.NewRequest("POST", "http://127.0.0.1:8080/tsf.test.helloworld.Greeter/SayHello", bytes.NewReader([]byte(content)))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("resp status code not 200")
	}
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if len(result) <= 0 {
		return fmt.Errorf("invalid resp")
	}
	var message struct {
		Message string `json:"message"`
	}
	err = json.Unmarshal([]byte(result), &message)
	return err
}
