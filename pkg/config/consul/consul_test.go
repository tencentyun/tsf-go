package consul

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/tencentyun/tsf-go/pkg/config"
	"gopkg.in/yaml.v3"
)

func TestMain(m *testing.M) {
	m.Run()

}

const testContent1 = `destList:
  - destId: route-2123123`

const testContent2 = `destList:
  - destId: route-2123123
  - destId: route-wahaha`

func TestSimpleConfig(t *testing.T) {
	err := set("com/tencent/tsf", []byte(testContent1))
	if err != nil {
		t.Logf("setConsul com/tencent/tsf failed!err:=%v", err)
		t.FailNow()
	}
	config := New(&Config{
		Address: "127.0.0.1:8500",
	})
	watcher := config.Subscribe("com/tencent/tsf")

	checkConfig(t, watcher, testContent1)

	err = set("com/tencent/tsf", []byte(testContent2))
	if err != nil {
		t.Logf("setConsul com/tencent/tsf failed!err:=%v", err)
		t.FailNow()
	}
	checkConfig(t, watcher, testContent2)
	err = deleteKey("com/tencent/tsf")
	if err != nil {
		t.Logf("delete com/tencent/tsf failed!err:=%v", err)
		t.FailNow()
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	specs, err := watcher.Watch(ctx)
	if err != nil {
		t.Logf("watch com/tencent/tsf failed!err:=%v", err)
		t.FailNow()
	}
	if specs != nil {
		t.Logf("watch com/tencent/tsf msut be nil")
		t.FailNow()
	}
	err = set("com/tencent/tsf", []byte(testContent2))
	if err != nil {
		t.Logf("setConsul com/tencent/tsf failed!err:=%v", err)
		t.FailNow()
	}
	checkConfig(t, watcher, testContent2)
}

type checkData struct {
	DestList []struct {
		DestId string `yaml:"destId"`
	} `yaml:"destList"`
}

func checkConfig(t *testing.T, watcher config.Watcher, expect string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	specs, err := watcher.Watch(ctx)
	if err != nil {
		t.Logf("watch com/tencent/tsf failed!err:=%v", err)
		t.FailNow()
	}
	var res checkData
	err = specs[0].Data.Unmarshal(&res)
	if err != nil {
		t.Logf("Unmarshal com/tencent/tsf failed!err:=%v", err)
		t.FailNow()
	}
	var check checkData
	err = yaml.Unmarshal([]byte(expect), &check)
	if err != nil {
		t.Logf("Unmarshal expect(%s) failed!err:=%v", expect, err)
		t.FailNow()
	}
	if !reflect.DeepEqual(check, res) {
		t.Logf("DeepEqual check(%+v) res(%+v) failed!not euqal", check, res)
		t.FailNow()
	}
}

func deleteKey(key string) error {
	client := http.Client{Timeout: time.Second * 2}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:8500/v1/kv/%s", key), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("response status (%d) not 200", resp.StatusCode)
	}
	return nil
}

func set(key string, value []byte) error {
	client := http.Client{Timeout: time.Second * 2}
	req, err := http.NewRequest("PUT", fmt.Sprintf("http://127.0.0.1:8500/v1/kv/%s", key), bytes.NewReader(value))
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("response status (%d) not 200", resp.StatusCode)
	}
	return nil
}
