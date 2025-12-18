package models

type CheckRequest struct {
	Key string `json:"key"`
}

type CheckResponse struct {
	Allowed bool `json:"allowed"`
}
