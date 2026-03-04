package lingo


type ValueError struct {
	Message string
}


func (v * ValueError) Error() string {
	return v.Message
}

type RuntimeError struct {
	Message string
}

func (r * RuntimeError) Error() string {
	return r.Message
}