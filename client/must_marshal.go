package client

import "encoding/json"

func mustMarshal(v interface{}) []byte {
	res, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return res
}
