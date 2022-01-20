package client

import "encoding/json"

func mustMarshal(v interface{}) []byte {
	res, _ := json.Marshal(v)
	return res
}
