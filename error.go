package lingo

// ValueError represents a validation or bad request error.
type ValueError struct {
	Message string
}

func (v *ValueError) Error() string {
	return v.Message
}

// RuntimeError represents a server-side or network error.
// StatusCode contains the HTTP status code when available, or 0 otherwise.
type RuntimeError struct {
	Message    string
	StatusCode int
}

func (r *RuntimeError) Error() string {
	return r.Message
}
