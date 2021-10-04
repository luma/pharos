package protocol

type Command string

const (
	QUIT Command = "QUIT"
	PING Command = "PING"
	SET  Command = "SET"
	GET  Command = "GET"
)

type ResponseType string

const (
	RespPong   ResponseType = "PONG"
	RespOk     ResponseType = "OK"
	RespGet    ResponseType = "GET"
	RespErr    ResponseType = "ERR"
	RespUpdate ResponseType = "UPDATE"
)
