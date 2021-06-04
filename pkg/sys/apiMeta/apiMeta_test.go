package apiMeta

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	str := "H4sIAAAAAAAAAE1OzQrCMAx+lZLzbNXjbiIeBBFBXyDWuBa2djTRHmTvbnQehFzy/b+AK3YdFWhhbZfQQEz3DO0LbsS+xFFiTspdQmSjJ9lwyNXgGM2/ooEnFZ61K7v8JkmUnvQ/zw1mc9or6nMS9KIVUwN99JSYPn0Jh494M6IPZOYtj9IrFETG1rlaq8Uva3Pp3M/K7rDf7o7n3UItNsjQw6TBV2Q6oQS1u0o3XAzEjB3B9AYl/e0q8gAAAA=="
	api, err := Decode(str)
	if err != nil {
		t.Fatalf("decode failed!err:=%v", err)
	}
	fmt.Println("api", api)
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
