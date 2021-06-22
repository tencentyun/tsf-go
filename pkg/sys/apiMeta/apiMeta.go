package apiMeta

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Service struct {
	PackageName string
	ServiceName string
	Paths       []Path
}

type Path struct {
	Method   string
	FullName string
}

type API struct {
	Paths map[string]map[string]Response `json:"paths"`
}

type Response struct {
	Codes map[string]ResponseEntity `json:"responses"`
}

type ResponseEntity struct {
	Schema      map[string]string `json:"schema"`
	Description string            `json:"description"`
}

func Encode(api *API) (res string, err error) {
	if api == nil {
		return
	}
	str, err := json.Marshal(api)
	if err != nil {
		return
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	n, err := zw.Write([]byte(str))
	if err != nil {
		return
	}
	if n <= 0 {
		err = fmt.Errorf("encode api err!")
		return
	}
	err = zw.Close()
	if err != nil {
		return
	}
	res = base64.StdEncoding.EncodeToString(buf.Bytes())
	return
}

func Decode(s string) (api *API, err error) {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return
	}
	r, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return
	}
	res, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}
	fmt.Println("result:", string(res))
	api = new(API)
	err = json.Unmarshal(res, api)
	return
}

func GenApiMeta(serDesc map[string]*Service) *API {
	api := API{
		Paths: make(map[string]map[string]Response),
	}
	for _, desc := range serDesc {
		for _, path := range desc.Paths {
			resp := Response{Codes: make(map[string]ResponseEntity)}
			resp.Codes["200"] = ResponseEntity{Schema: make(map[string]string), Description: "OK"}
			methods := make(map[string]Response)
			methods[path.Method] = resp
			api.Paths[path.FullName] = methods
		}
	}
	return &api
}
