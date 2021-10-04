package protocol

import (
	"bytes"
	"fmt"
	"io"
)

var (
	OkTerminal = []byte("OK\r\n")
	Terminal   = []byte("\r\n")
)

func WriteOk(w io.Writer, requestID RequestID) error {
	_, err := w.Write(PrependRequestID(OkTerminal, requestID))
	return err
}

func WriteString(w io.Writer, requestID RequestID, s string) error {
	b := append([]byte(s), '\r', '\n')
	_, err := w.Write(PrependRequestID(b, requestID))
	return err
}

func WriteLines(w io.Writer, requestID RequestID, ss ...[]byte) error {
	if len(ss) == 0 {
		return nil
	}

	lines := make([][]byte, 0, len(ss))
	lines = append(lines, PrependRequestID(ss[0], requestID))
	lines = append(lines, ss[1:]...)

	b := bytes.Join(lines, Terminal)
	b = append(b, '\r', '\n')

	_, err := w.Write(b)
	return err
}

func WriteError(w io.Writer, requestID RequestID, errMsg string) error {
	b := []byte(fmt.Sprintf("ERR %s\r\n", errMsg))
	_, err := w.Write(PrependRequestID(b, requestID))
	return err
}

func PrependRequestID(data []byte, requestID RequestID) []byte {
	return append(requestID[:], data...)
}
