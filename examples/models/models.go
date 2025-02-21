package models

// Body is the response body.
type Body[T any] struct {
	Message string `json:"message"`
	Data    T      `json:"data"`
	Code    int    `json:"code"`
}

// Topic of a thread.
type Topic struct {
	Namespace string `json:"namespace"`
	Topic     string `json:"topic"`
	Private   bool   `json:"private"`
}
