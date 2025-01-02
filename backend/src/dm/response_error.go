package dm

type ResponseError struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
	Body  any    `json:"body"`
}
