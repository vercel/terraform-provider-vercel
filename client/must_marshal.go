package client

import "encoding/json"

// mustMarshal is a helper to remove unnecessary error checking when marshaling a Go
// struct to json. There are only a few instances where marshaling can fail, and they
// are around the shape of the data. e.g. if a struct contains a channel, then it cannot
// be marshaled. As our structs are known ahead of time and are all safe to marshal,
// this simplifies the error checking process.
func mustMarshal(v interface{}) []byte {
	res, _ := json.Marshal(v)
	return res
}
