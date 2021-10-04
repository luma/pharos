package client

import (
	"context"
	"encoding/binary"
	"math"
	"net"
	"sync"

	"github.com/luma/pharos/protocol"
	"go.uber.org/zap"
)

type Update struct {
	Key   string
	Value []byte
}

type Conn struct {
	ctx context.Context

	conn *net.TCPConn

	updateChan chan *Update

	respMu    sync.RWMutex
	respChans map[protocol.RequestID]chan *protocol.Response

	idMu      sync.Mutex
	requestId uint32

	log *zap.Logger
}

func New(log *zap.Logger) *Conn {
	return &Conn{
		log:        log,
		updateChan: make(chan *Update, 255),
		respChans:  make(map[protocol.RequestID]chan *protocol.Response),
	}
}

func (c *Conn) Connect(ctx context.Context, addr string) error {
	c.ctx = ctx

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}

	c.conn = conn.(*net.TCPConn)

	return nil
}

func (c *Conn) Disconnect() error {
	// TODO(rolly) mark us as disconnected and have all methods that make command requests return disconnected errors
	// TODO(rolly) tell the read loop to terminate and wait until it does

	return c.conn.Close()
}

func (c *Conn) UpdateChan() <-chan *Update {
	return c.updateChan
}

func (c *Conn) Quit(ctx context.Context) error {
	reqID, respChan := c.createResponseChan()
	defer c.destroyResponseChan(reqID)

	err := protocol.WriteString(c.conn, reqID, "QUIT")
	if err != nil {
		return err
	}

	select {
	case resp := <-respChan:
		return resp.ErrorOrNil()

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Conn) Ping(ctx context.Context) error {
	reqID, respChan := c.createResponseChan()
	defer c.destroyResponseChan(reqID)

	err := protocol.WriteString(c.conn, reqID, "PING")
	if err != nil {
		return err
	}

	select {
	case resp := <-respChan:
		return resp.ErrorOrNil()

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Conn) Set(ctx context.Context, key string, value []byte) error {
	reqID, respChan := c.createResponseChan()
	defer c.destroyResponseChan(reqID)

	err := protocol.WriteLines(c.conn, reqID, protocol.PrefixSet, []byte(key), value)
	if err != nil {
		return err
	}

	select {
	case resp := <-respChan:
		return resp.ErrorOrNil()

	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Conn) readLoop() {
	log := c.log.Named("readLoop")

	for {
		select {
		case <-c.ctx.Done():
			log.Info("Context cancelled, exiting...")
			return

		default:
			// TODO(rolly) probably want to SetDeadline on the reads...

			// Parse command responses and
			resp, err := protocol.ReadResponse(c.conn)
			if err != nil {
				log.Warn("Failed to read server response", zap.Error(err))
				continue
			}

			if resp.Type == protocol.RespUpdate {
				// Handle responses that indicate keys were updated
				c.updateChan <- &Update{
					Key:   string(resp.Args[0].([]byte)),
					Value: resp.Value,
				}
				continue
			}

			// Handle responses to our requests
			c.sendToResponseChan(resp.RequestID, resp)
		}
	}
}

func (c *Conn) createResponseChan() (protocol.RequestID, <-chan *protocol.Response) {
	reqID := c.getNextRequestID()
	respChan := make(chan *protocol.Response, 1)

	c.respMu.Lock()
	c.respChans[reqID] = respChan
	c.respMu.Unlock()

	return reqID, respChan
}

func (c *Conn) sendToResponseChan(reqID protocol.RequestID, resp *protocol.Response) {
	c.respMu.Lock()
	respChan, ok := c.respChans[reqID]
	c.respMu.Unlock()

	if !ok {
		return
	}

	respChan <- resp
	c.destroyResponseChan(reqID)
}

func (c *Conn) destroyResponseChan(reqID protocol.RequestID) {
	c.respMu.Lock()
	respChan, ok := c.respChans[reqID]
	if ok {
		close(respChan)
		delete(c.respChans, reqID)
	}
	c.respMu.Unlock()
}

func (c *Conn) getNextRequestID() protocol.RequestID {
	var requestID uint32

	c.idMu.Lock()
	if c.requestId < math.MaxUint32-1 {
		c.requestId += 1
	} else {
		// Wrap around instead of overflowing
		c.requestId = 0
	}

	requestID = c.requestId
	c.idMu.Unlock()

	var reqID protocol.RequestID
	binary.LittleEndian.PutUint32(reqID[:], requestID)
	return reqID
}
