package protocol

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

var (
	ErrUnknownCommand          = errors.New("Unknown command could not be parsed")
	ErrRequestTooShort         = errors.New("Request is malformed, it appears to be too short")
	ErrRequestUnexpectedEOF    = errors.New("Request is malformed, received EOF before parsing a full command")
	ErrRequestMissingSetSpace  = errors.New("Set command is malformed, it appears to be missing a space between SET and the key")
	ErrResponseMissingErrSpace = errors.New("Err command response is malformed, it appears to be missing a space between ERR and the error messsage")

	PrefixQuit = []byte("QUIT")
	PrefixPing = []byte("PING")
	PrefixGet  = []byte("GET")
	PrefixSet  = []byte("SET")
	PrefixPong = []byte("PONG")
	PrefixOk   = []byte("OK")
	PrefixErr  = []byte("ERR")

	// PrefixUpdate starts the first line of every update from the server
	PrefixUpdate = []byte("*")
)

// ReadRequest reads bytes from the provided Reader and attempts to parse them
// as a Pharos request command.
//
// To avoid denial of service attacks, the provided bufio.Reader
// should be reading from an io.LimitReader or similar Reader to bound
// the size of responses.
func ReadRequest(data io.Reader) (req Request, err error) {
	r := bufio.NewReader(data)

	// Read the Command
	rawReq, err := r.ReadBytes('\n')
	if err != nil {
		// TODO(rolly)
		// This could be handled better. It's possible that we don't have a '\n'
		// yet as we haven't received enough from the client. In this case we
		// would accumulate more until we have a '\n' or we reach should safe
		// limit on buffer size.
		//
		// We should handle the above case and only return for other cases or
		// if we hit our buffer limit
		return nil, err
	}

	if len(rawReq) < 9 {
		return nil, ErrRequestTooShort
	}

	var requestID RequestID
	copy(requestID[:], rawReq[:4])

	// Strip off the request id and the final '\n'
	rawCommand := rawReq[4 : len(rawReq)-1]

	// Parse the command
	switch {
	case bytes.HasPrefix(rawCommand, PrefixQuit):
		req := &QuitRequest{requestID: requestID}
		return req, nil

	case bytes.HasPrefix(rawCommand, PrefixPing):
		req := &PingRequest{requestID: requestID}
		return req, nil

	case bytes.HasPrefix(rawCommand, PrefixGet):
		req := &GetRequest{requestID: requestID}

		if rawCommand[3] != ' ' {
			// There should be a space delimiting the SET from it's key
			return nil, fmt.Errorf("Failed to parse '%s': %w",
				string(rawCommand), ErrRequestMissingSetSpace)
		}

		// Read key to get
		req.Key = RemoveTrailingCR(rawCommand[4:])

		return req, nil

	case bytes.HasPrefix(rawCommand, PrefixSet):
		req := &SetRequest{requestID: requestID}

		if rawCommand[3] != ' ' {
			// There should be a space delimiting the SET from it's key
			return nil, fmt.Errorf("Failed to parse '%s': %w",
				string(rawCommand), ErrRequestMissingSetSpace)
		}

		// Read key to set
		req.Key = RemoveTrailingCR(rawCommand[4:])

		// Ready key value
		value, err := r.ReadBytes('\n')

		if err != nil {
			// TODO(rolly)
			// This could be handled better. It's possible that we don't have a '\n'
			// yet as we haven't received enough from the client. In this case we
			// would accumulate more until we have a '\n' or we reach should safe
			// limit on buffer size.
			//
			// We should handle the above case and only return for other cases or
			// if we hit our buffer limit
			return nil, err
		}

		req.Value = RemoveTrailingCR(value[:len(value)-1])

		return req, nil

	default:
		return nil, fmt.Errorf("Failed to parse '%s': %w",
			string(rawCommand), ErrUnknownCommand)
	}
}

// ReadResponse reads bytes from the provided Reader and attempts to parse them
// as a Pharos response.
//
// To avoid denial of service attacks, the provided bufio.Reader
// should be reading from an io.LimitReader or similar Reader to bound
// the size of responses.
func ReadResponse(data io.Reader) (resp *Response, err error) {
	r := bufio.NewReader(data)

	// Read the Command
	rawResp, err := r.ReadBytes('\n')
	if err != nil {
		// TODO(rolly)
		// This could be handled better. It's possible that we don't have a '\n'
		// yet as we haven't received enough from the client. In this case we
		// would accumulate more until we have a '\n' or we reach should safe
		// limit on buffer size.
		//
		// We should handle the above case and only return for other cases or
		// if we hit our buffer limit
		return nil, err
	}

	if len(rawResp) < 9 {
		return nil, ErrRequestTooShort
	}

	if rawResp[0] == PrefixUpdate[0] {
		// This is a update pushed from the server, not a response to
		// a client request.
		key := rawResp[1 : len(rawResp)-1]

		// Ready Get response value
		value, err := r.ReadBytes('\n')

		if err != nil {
			// TODO(rolly)
			// This could be handled better. It's possible that we don't have a '\n'
			// yet as we haven't received enough from the client. In this case we
			// would accumulate more until we have a '\n' or we reach should safe
			// limit on buffer size.
			//
			// We should handle the above case and only return for other cases or
			// if we hit our buffer limit
			return nil, err
		}

		resp := &Response{
			Type:  RespUpdate,
			Args:  []interface{}{key},
			Value: RemoveTrailingCR(value[:len(value)-1]),
		}

		return resp, nil
	}

	var requestID RequestID
	copy(requestID[:], rawResp[:4])

	// Strip off the request id and the final '\n'
	rawCommand := rawResp[4 : len(rawResp)-1]

	// Parse the command
	switch {
	case bytes.HasPrefix(rawCommand, PrefixPong):
		resp := &Response{Type: RespPong, RequestID: requestID}
		return resp, nil

	case bytes.HasPrefix(rawCommand, PrefixOk):
		resp := &Response{Type: RespOk, RequestID: requestID}
		return resp, nil

	case bytes.HasPrefix(rawCommand, PrefixGet):
		// Ready Get response value
		value, err := r.ReadBytes('\n')

		if err != nil {
			// TODO(rolly)
			// This could be handled better. It's possible that we don't have a '\n'
			// yet as we haven't received enough from the client. In this case we
			// would accumulate more until we have a '\n' or we reach should safe
			// limit on buffer size.
			//
			// We should handle the above case and only return for other cases or
			// if we hit our buffer limit
			return nil, err
		}

		resp := &Response{
			Type:      RespGet,
			RequestID: requestID,
			Value:     RemoveTrailingCR(value[:len(value)-1]),
		}

		return resp, nil

	case bytes.HasPrefix(rawCommand, PrefixErr):
		// <reqID>ERR <errMessage>\r\n

		if rawCommand[3] != ' ' {
			// There should be a space delimiting the ERR from it's message
			return nil, fmt.Errorf("Failed to parse '%s': %w",
				string(rawCommand), ErrResponseMissingErrSpace)
		}

		resp := &Response{
			Type:      RespErr,
			RequestID: requestID,
			Args: []interface{}{
				errors.New(string(rawCommand[4:])),
			},
		}

		return resp, nil

	default:
		return nil, fmt.Errorf("Failed to parse '%s': %w",
			string(rawCommand), ErrUnknownCommand)
	}
}

func RemoveTrailingCR(data []byte) []byte {
	if data[len(data)-1] == '\r' {
		// Remove the optional trailing \r
		return data[:len(data)-1]
	}

	return data
}
