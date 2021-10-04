package transport_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"time"

	"github.com/luma/pharos/storage"
	"github.com/luma/pharos/transport"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("transport", func() {
	Describe("TCP", func() {
		It("listens on the desired port", func() {
			tcp := makeTCPServer("")

			defer func() {
				Expect(tcp.Close()).To(Succeed())
			}()

			conn, err := net.Dial("tcp", "0.0.0.0:6682")
			Expect(err).To(Succeed())
			conn.Close()
		})

		// It("will close client connections when they QUIT", func() {
		// 	tcp := makeTCPServer("")

		// 	conn, err := net.Dial("tcp", "0.0.0.0:6682")
		// 	Expect(err).To(Succeed())

		// 	defer func() {
		// 		conn.Close()
		// 		Expect(tcp.Close()).To(Succeed())
		// 	}()

		// 	_, err = conn.Write([]byte("1234QUIT\n"))
		// 	Expect(err).To(Succeed())

		// 	response, err := bufio.NewReader(conn).ReadBytes('\n')
		// 	Expect(err).To(Succeed())

		// 	Expect(string(response)).To(Equal("1234OK\r\n"))

		// 	waitForClose(conn)
		// })

		// It("will respond with PONG when the client sends PING", func() {
		// 	tcp := makeTCPServer("")

		// 	conn, err := net.Dial("tcp", "0.0.0.0:6682")
		// 	Expect(err).To(Succeed())

		// 	defer func() {
		// 		conn.Close()
		// 		Expect(tcp.Close()).To(Succeed())
		// 	}()

		// 	_, err = conn.Write([]byte("1234PING\n"))
		// 	Expect(err).To(Succeed())

		// 	response, err := bufio.NewReader(conn).ReadBytes('\n')
		// 	Expect(err).To(Succeed())

		// 	Expect(string(response)).To(Equal("1234PONG\r\n"))

		// 	waitForClose(conn)
		// })

		// Describe("SET command", func() {
		// 	It("will respond with OK when the SET command suceeds", func() {
		// 		tcp := makeTCPServer()

		// 		conn, err := net.Dial("tcp", "0.0.0.0:6682")
		// 		Expect(err).To(Succeed())

		// 		defer func() {
		// 			conn.Close()
		// 			Expect(tcp.Close()).To(Succeed())
		// 		}()

		// 		_, err = conn.Write([]byte("1234SET foo\nbar\n"))
		// 		Expect(err).To(Succeed())

		// 		response, err := bufio.NewReader(conn).ReadBytes('\n')
		// 		Expect(err).To(Succeed())

		// 		Expect(string(response)).To(Equal("1234OK\r\n"))

		// 		waitForClose(conn)
		// 	})

		// 	It("writes the new value of the key to the store", func() {
		// 		tcp := makeTCPServer()

		// 		conn, err := net.Dial("tcp", "0.0.0.0:6682")
		// 		Expect(err).To(Succeed())

		// 		defer func() {
		// 			conn.Close()
		// 			Expect(tcp.Close()).To(Succeed())
		// 		}()

		// 		_, err = conn.Write([]byte("1234SET foo\nbar\n"))
		// 		Expect(err).To(Succeed())

		// 		response, err := bufio.NewReader(conn).ReadBytes('\n')
		// 		Expect(err).To(Succeed())

		// 		Expect(string(response)).To(Equal("1234OK\r\n"))

		// 		value, err := tcp.Store().Get(context.Background(), []byte("foo"))
		// 		Expect(err).To(Succeed())
		// 		Expect(string(value)).To(Equal(`"bar"`))

		// 		waitForClose(conn)
		// 	})
		// })

		Describe("GET command", func() {
			It("returns the current value of a key", func() {
				tcp := makeTCPServer(`{"foo":"bar"}`)

				conn, err := net.Dial("tcp", "0.0.0.0:6682")
				Expect(err).To(Succeed())

				defer func() {
					conn.Close()
					Expect(tcp.Close()).To(Succeed())
				}()

				_, err = conn.Write([]byte("1234GET foo\n"))
				Expect(err).To(Succeed())

				var response []byte
				_, err = io.ReadFull(bufio.NewReader(conn), response)
				Expect(err).To(Succeed())
				Expect(string(response)).To(Equal("1234GET\r\n\"bar\"\r\n"))

				// response, err := bufio.NewReader(conn).ReadBytes('\n')
				// Expect(err).To(Succeed())
				// Expect(string(response)).To(Equal("1234GET\r\n"))

				// response, err = bufio.NewReader(conn).ReadBytes('\n')
				// Expect(err).To(Succeed())
				// Expect(string(response)).To(Equal(`"bar"\r\n`))

				waitForClose(conn)
			})
		})
	})
})

func waitForClose(conn net.Conn) {
	// Wait to our client to be disconnected by the server
	timeout := time.After(30 * time.Second)

waitForClose:
	for {
		select {
		case <-timeout:
			Fail("The client was never closed by the server")
			break waitForClose

		case <-time.After(10 * time.Millisecond):
			// This '1' business is because zero-width reads return
			// immediately and do nothing, our test needs to actually
			// attempt a read
			one := make([]byte, 1)
			Expect(conn.SetReadDeadline(time.Now())).To(Succeed())
			_, err := conn.Read(one)

			timeoutErr, ok := err.(net.Error)
			if ok {
				Expect(timeoutErr.Timeout()).To(BeTrue())
				break waitForClose
			}
		}
	}
}

func makeTCPServer(restore string) *transport.TCP {
	store := storage.NewInmemoryStore()
	if restore != "" {
		Expect(store.Restore([]byte(restore))).To(Succeed())
	}

	log, err := zap.NewDevelopment()
	Expect(err).To(Succeed())

	tcp := transport.NewTCP(transport.Options{
		Log:          log,
		NumListeners: 1,
		Port:         6682,

		// TODO(rolly) Reuseport should default to true
		Reuseport: true,

		Store: store,
	})

	err = tcp.Start(context.Background())
	Expect(err).To(Succeed())

	// Wait for the TCP server to be listening.
	// TODO(rolly) this is stupid, either make sure `tcp.Start()` does not
	//						 return until the server is listening or provide a test
	//						 helper that retries until a connection is achieved or a
	//						 timeout is hit.
	time.Sleep(100 * time.Millisecond)

	return tcp
}

func readLine(conn net.Conn) ([]byte, error) {
	r := bufio.NewReader(conn)
	var line []byte

	for {
		chunk, more, err := r.ReadLine()
		if err != nil {
			return nil, err
		}

		// Avoid the copy if the first call produced a full line.
		if line == nil && !more {
			return chunk, nil
		}

		line = append(line, chunk...)

		if !more {
			break
		}
	}

	return line, nil
}
