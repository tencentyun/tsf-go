package apiMeta

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	str := "H4sIAAAAAAAAAE1OvQ7CIBB+FXJzBXXs1pgOTYxpUl8AEQsJBQKnDE3f3WvrYHLLff8z5CLHUSeo4cyPUIH1rwD1DE+dVbIRbfDE3Y3NjA4DyyYUJqNl/4oKPjrlXXvixy0JLTpN/7A3sKbvCFXBo1RIFUsFzirts177vJxWcROlMprtW97JEWQQYy1EKYXLjeUhjeJnzeLaXdrb0B7Iwg1ODhYKfsise4mG7AKWL42lUUXmAAAA"
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
