package protocol_test

import (
	"bytes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/luma/pharos/protocol"
)

var _ = Describe("Parsing/ Writer", func() {
	var reqID protocol.RequestID
	copy(reqID[:], []byte("1234"))

	Describe("WriteOk", func() {
		It("includes the request ID as a prefix", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteOk(w, reqID)).To(Succeed())
			Expect(w.String()).To(HavePrefix("1234"))
		})

		It("ends in \r\n", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteOk(w, reqID)).To(Succeed())
			Expect(w.String()).To(HaveSuffix("\r\n"))
		})

		It("include OK", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteOk(w, reqID)).To(Succeed())
			Expect(w.String()).To(Equal("1234OK\r\n"))
		})
	})

	Describe("WriteString", func() {
		It("includes the request ID as a prefix", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteString(w, reqID, "resp")).To(Succeed())
			Expect(w.String()).To(HavePrefix("1234"))
		})

		It("ends in \r\n", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteString(w, reqID, "resp")).To(Succeed())
			Expect(w.String()).To(HaveSuffix("\r\n"))
		})

		It("include the response string", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteString(w, reqID, "resp")).To(Succeed())
			Expect(w.String()).To(Equal("1234resp\r\n"))
		})
	})

	Describe("WriteStrings", func() {
		It("includes the request ID as a prefix", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteLines(w, reqID, []byte("key"), []byte("value"))).To(Succeed())
			Expect(w.String()).To(HavePrefix("1234"))
		})

		It("ends in \r\n", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteLines(w, reqID, []byte("key"), []byte("value"))).To(Succeed())
			Expect(w.String()).To(HaveSuffix("\r\n"))
		})

		It("include the response string", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteLines(w, reqID, []byte("key"), []byte("value"))).To(Succeed())
			Expect(w.String()).To(Equal("1234key\r\nvalue\r\n"))
		})
	})

	Describe("WriteError", func() {
		It("includes the request ID as a prefix", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteError(w, reqID, "errMessage")).To(Succeed())
			Expect(w.String()).To(HavePrefix("1234"))
		})

		It("ends in \r\n", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteError(w, reqID, "errMessage")).To(Succeed())
			Expect(w.String()).To(HaveSuffix("\r\n"))
		})

		It("include the ERR response code and the error string", func() {
			w := bytes.NewBuffer([]byte{})

			Expect(protocol.WriteError(w, reqID, "errMessage")).To(Succeed())
			Expect(w.String()).To(Equal("1234ERR errMessage\r\n"))
		})
	})
})
