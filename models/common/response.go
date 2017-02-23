package common

type Response struct {
	Count    int         `json:"count"`
	Next     interface{} `json:"next"`
	Previous interface{} `json:"previous"`
	Data     interface{} `json:"data"`
}
