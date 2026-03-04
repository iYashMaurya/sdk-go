package lingo


type valueError struct {
	Message string
}


func (v * valueError) Error() string {
	return v.Message
}