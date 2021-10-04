package protocol

type Response struct {
	Type      ResponseType
	RequestID RequestID
	Args      []interface{}
	Value     []byte
}

// ErrorOrNil returns an error if the response contains an error. Otherwise it
// returns nil.
func (r *Response) ErrorOrNil() error {
	if r.Type == RespErr {
		return r.Args[0].(error)
	}

	return nil
}
