package apiMeta

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	str := "H4sIAAAAAAAA/xzLwQrCMBAE0H/Zs6TFY35AwYMHvyC0IynEbthZEAn9d9nO5cEwM6QXr5Q8ZHK+k4OeKlrTr1pb080Ah02v8rtHG8Ou9NDArjtxvq/zHHCp+BTJ47jICi62dd90lyzPhxxn/gEAAP//BbRRR3MAAAA="
	api, err := Decode(str)
	if err != nil {
		t.Fatalf("decode failed!err:=%v", err)
	}
	fmt.Println(api)
}

func TestEncodeDecode(t *testing.T) {
	api := API{
		Paths: map[string]map[string]Response{},
	}
	resp := Response{Codes: make(map[string]ResponseEntity)}
	resp.Codes["200"] = ResponseEntity{Schema: make(map[string]string), Description: "OK"}
	paths := make(map[string]Response)
	paths["post"] = resp
	api.Paths["sayHello2233"] = paths

	enStr, err := Encode(&api)
	if err != nil {
		t.Fatalf("encode,err:=%v", err)
	}
	apiDe, err := Decode(enStr)
	if err != nil {
		t.Fatalf("decode,err:=%v", err)
	}
	if !reflect.DeepEqual(*apiDe, api) {
		t.Fatalf("api(%v) not eqaul apiDe(%v)", api, *apiDe)
	}
}
