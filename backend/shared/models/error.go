package models

type ErrorObject struct {
	Error   string `json:"error"`
	Details string `json:"details"`
}
