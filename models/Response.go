package models

// Response is the model for response
// collection
type Response struct {
	Count   int `bson:"count" json:"count"`
	Results interface{}        `bson:"results" json:"results"`
}