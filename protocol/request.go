package protocol

type RequestID [4]byte

func (r RequestID) String() string {
	return string(r[:])
}

type Request interface {
	GetRequestID() RequestID
	GetCommand() Command
}

type QuitRequest struct {
	requestID RequestID
}

func (q *QuitRequest) GetRequestID() RequestID {
	return q.requestID
}

func (q *QuitRequest) GetCommand() Command {
	return QUIT
}

type PingRequest struct {
	requestID RequestID
}

func (q *PingRequest) GetRequestID() RequestID {
	return q.requestID
}

func (q *PingRequest) GetCommand() Command {
	return PING
}

type SetRequest struct {
	requestID RequestID
	Key       []byte
	Value     []byte
}

func (q *SetRequest) GetRequestID() RequestID {
	return q.requestID
}

func (q *SetRequest) GetCommand() Command {
	return SET
}

type GetRequest struct {
	requestID RequestID
	Key       []byte
}

func (q *GetRequest) GetRequestID() RequestID {
	return q.requestID
}

func (q *GetRequest) GetCommand() Command {
	return GET
}

var _ Request = (*QuitRequest)(nil)
var _ Request = (*PingRequest)(nil)
var _ Request = (*SetRequest)(nil)
var _ Request = (*GetRequest)(nil)
