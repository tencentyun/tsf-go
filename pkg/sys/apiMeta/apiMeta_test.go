package apiMeta

import (
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	str := "H4sIAAAAAAAA/6xVwY7aMBC95ytGbo8rQPTGbU/d3qqyVQ/VqjLJBLxybO94vFtU5d8rG0KcAIWK5RSPZ57fG88zfwoA4d/keo0kFiDmk5m4izFlaisWEPcBBCvWGPc3qLV9s6SriSPLNiUDiFckr6yJKftPMJbBI4sCoE2QLNdeLOBnqtgBAwgjm4T8mRAZSaR4WwA8pSJfbrDBvk48PD5+7U6N38tu8SP7WooDQGmNDwME6ZxWpWRlzfTZW9PnOrJVKK/MlbzxfYemWWf2UqZLuX2I0UNWLLOeszWAsA4pHfClyvrw61B816cSemeNRz9AABDz2WwUAhAV+pKU4/293IMPZYne10FDhzTJ4FNR6rc8AgMQHwnriPNhWmGtjIq4PpOd2H5Dp7diUNpmqzY/TVRYy6D5MnMDweBvhyVjBUhk6f0EkCuXLDn4f7AuTvAXTpJs4lX147L7jcR0A76y1XZMVplzO4QvQRHGkWAK+N639BLQ8zWKnzLFAwPvYwPbpoIih2gP7s/o9K45OT2ZW3jrUu/s6hlLPvQoGtUhsRo5QTTovVzj2B4djGdSZt1zbYdc786Q2jXrBlr7EbiBU3psV6G+Nzf1J+Z/J301l+zyX6UOl0QM3F1baiSn+d4yXhLY2/AGeaWtzlJUhjH+yZ3hqAx/mp9W/r9DlZVWyFLpo+e6K5VEcuh8oRibcf5ZW+dTcfq9OjZj0RZ/AwAA//9aGDaN9QcAAA=="
	_, err := Decode(str)
	if err != nil {
		t.Fatalf("decode failed!err:=%v", err)
	}
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
