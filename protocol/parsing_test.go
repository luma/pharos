package protocol_test

import (
	"bytes"
	"errors"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/luma/pharos/protocol"
)

var _ = Describe("Parsing", func() {
	Describe("ReadRequest()", func() {
		var expectedRequestID protocol.RequestID
		copy(expectedRequestID[:], []byte("1234"))

		It("returns an error if the reader cannot find a newline", func() {
			data := bytes.NewReader([]byte("I have no new line"))
			_, err := protocol.ReadRequest(data)
			Expect(err).To(MatchError(io.EOF))
		})

		It("returns an error if the data is too short to be a valid request", func() {
			data := bytes.NewReader([]byte("hello\n"))
			_, err := protocol.ReadRequest(data)
			Expect(err).To(MatchError(protocol.ErrRequestTooShort))

			data = bytes.NewReader([]byte("1234\n"))
			_, err = protocol.ReadRequest(data)
			Expect(err).To(MatchError(protocol.ErrRequestTooShort))
		})

		It("returns an error if the command is unknown", func() {
			data := bytes.NewReader([]byte("1234EVIL\n"))
			_, err := protocol.ReadRequest(data)
			Expect(errors.Is(err, protocol.ErrUnknownCommand)).To(BeTrue())
		})

		It("parses a valid QUIT command", func() {
			data := bytes.NewReader([]byte("1234QUIT\n"))
			req, err := protocol.ReadRequest(data)
			Expect(err).To(Succeed())
			Expect(req.GetRequestID()).To(Equal(expectedRequestID))
			Expect(req.GetCommand()).To(Equal(protocol.QUIT))
		})

		It("parses a valid PING command", func() {
			data := bytes.NewReader([]byte("1234PING\n"))
			req, err := protocol.ReadRequest(data)
			Expect(err).To(Succeed())
			Expect(req.GetRequestID()).To(Equal(expectedRequestID))
			Expect(req.GetCommand()).To(Equal(protocol.PING))
		})

		Describe("SET", func() {
			It("parses a valid SET command", func() {
				data := bytes.NewReader([]byte("1234SET key\nvalue\n"))
				req, err := protocol.ReadRequest(data)
				Expect(err).To(Succeed())
				Expect(req.GetRequestID()).To(Equal(expectedRequestID))
				Expect(req.GetCommand()).To(Equal(protocol.SET))

				setReq, ok := req.(*protocol.SetRequest)
				Expect(ok).To(BeTrue())

				Expect(setReq.Key).To(Equal([]byte("key")))
				Expect(setReq.Value).To(Equal([]byte("value")))
			})

			It("returns an error if there is not a space between the SET command and it's key", func() {
				data := bytes.NewReader([]byte("1234SETkey\nvalue\n"))
				_, err := protocol.ReadRequest(data)
				Expect(errors.Is(err, protocol.ErrRequestMissingSetSpace)).To(BeTrue())
			})

			It("returns an error if there is no final newline after the set value", func() {
				data := bytes.NewReader([]byte("1234SET key\nvalue"))
				_, err := protocol.ReadRequest(data)
				Expect(errors.Is(err, io.EOF)).To(BeTrue())
			})
		})

		Describe("GET", func() {
			It("parses a valid GET command", func() {
				data := bytes.NewReader([]byte("1234GET key\n"))
				req, err := protocol.ReadRequest(data)
				Expect(err).To(Succeed())
				Expect(req.GetRequestID()).To(Equal(expectedRequestID))
				Expect(req.GetCommand()).To(Equal(protocol.GET))

				getReq, ok := req.(*protocol.GetRequest)
				Expect(ok).To(BeTrue())

				Expect(getReq.Key).To(Equal([]byte("key")))
			})

			It("returns an error if there is not a space between the GET command and it's key", func() {
				data := bytes.NewReader([]byte("1234GETkey\n"))
				_, err := protocol.ReadRequest(data)
				Expect(errors.Is(err, protocol.ErrRequestMissingSetSpace)).To(BeTrue())
			})
		})
	})

	Describe("RemoveTrailingCR()", func() {
		It("does nothing if the data does not end in CR", func() {
			data := []byte("I am awesome data")
			Expect(protocol.RemoveTrailingCR(data)).To(Equal(data))
		})

		It("removes the trailling CR", func() {
			input := []byte("I am awesome data\r")
			output := []byte("I am awesome data")
			Expect(protocol.RemoveTrailingCR(input)).To(Equal(output))
		})
	})
})
