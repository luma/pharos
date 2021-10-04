package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	reuseport "github.com/kavu/go_reuseport"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/luma/pharos/protocol"
	"github.com/luma/pharos/storage"
)

const (
	UpdateBufferSize = 255
)

type Marshaler interface {
	Marshal() ([]byte, error)
}

type Update struct {
	Key  string
	Body Marshaler
}

type Write struct {
	Lines [][]byte
}

type TCP struct {
	cancel     context.CancelFunc
	stopWaiter sync.WaitGroup

	addr string

	numListeners int
	listeners    []*TCPListener

	store storage.Store

	mu       sync.Mutex
	doneChan chan struct{}

	log   *zap.Logger
	trace bool
}

func NewTCP(options Options) *TCP {
	numListeners := options.NumListeners

	if numListeners < 1 {
		numListeners = runtime.NumCPU()
	}

	return &TCP{
		addr:         net.JoinHostPort(options.Host, strconv.Itoa(options.Port)),
		numListeners: numListeners,
		listeners:    make([]*TCPListener, 0, options.NumListeners),
		doneChan:     make(chan struct{}),
		trace:        options.Trace,
		store:        options.Store,
		log:          options.Log,
	}
}

func (w *TCP) Start(parentCtx context.Context) error {
	ctx, cancel := context.WithCancel(parentCtx)
	w.cancel = cancel

	w.log.Info("Starting tcp listeners", zap.Int("count", w.numListeners))

	for i := 0; i < w.numListeners; i++ {
		w.startListener(ctx, w.addr)
	}

	return nil
}

func (t *TCP) Store() storage.Store {
	return t.store
}

func (w *TCP) startListener(ctx context.Context, addr string) {
	w.stopWaiter.Add(1)
	listener := NewTCPListener(
		ctx,
		addr,
		w.store,
		w.log.Named("listener").With(zap.Int("listener", len(w.listeners))),
	)

	w.listeners = append(w.listeners, &listener)

	go func() {
		defer w.stopWaiter.Done()

		if err := listener.Listen(); err != nil {
			// TODO(rolly) as any of the listeners can fail to listen, but we don't treat this as fatal,
			//             you can end up with less than the required amount of listeners running
			w.log.Error("Failed to listen", zap.Error(err))
		}
	}()
}

// Close immediately closes all active listeners and conenctions.
//
// For a graceful shutdown, use Shutdown()
func (w *TCP) Close() error {
	w.log.Info("Stopping TCP server")
	w.cancel()

	// Tell listeners to stop
	for _, listener := range w.listeners {
		listener.Close()
	}

	w.log.Info("WAITING FOR LISTENERS")
	w.stopWaiter.Wait()
	w.log.Info("LISTENERS STOPPED")

	return nil
}

func (w *TCP) closeDoneChan() {
	w.mu.Lock()
	defer w.mu.Unlock()

	select {
	case <-w.doneChan:
		// Already closed.
	default:
		close(w.doneChan)
	}
}

func (w *TCP) Shutdown(ctx context.Context) error {
	return nil
}

type TCPListener struct {
	ctx context.Context

	addr string
	log  *zap.Logger

	mu          sync.Mutex
	activeConns map[*TCPConn]struct{}

	writeQueues [](chan []byte)

	store storage.Store
}

func NewTCPListener(
	ctx context.Context,
	addr string,
	store storage.Store,
	log *zap.Logger,
) TCPListener {
	return TCPListener{
		ctx:         ctx,
		activeConns: make(map[*TCPConn]struct{}),
		writeQueues: make([](chan []byte), 0),
		addr:        addr,
		store:       store,
		log:         log,
	}
}

func (t *TCPListener) Close() error {
	// Close active connections
	// TODO(rolly) We're closing all connections here, not just active
	for conn := range t.activeConns {
		conn.Close()
		delete(t.activeConns, conn)
	}

	return nil
}

func (t *TCPListener) Listen() error {
	listener, err := reuseport.Listen("tcp", t.addr)
	if err != nil {
		return err
	}

	defer listener.Close()

	var (
		loopWaiter sync.WaitGroup
	)

	go func() {
		<-t.ctx.Done()

		t.log.Info("Draining reader/writer loops")
		loopWaiter.Wait()

		t.log.Info("Closing listener")
		if err := listener.Close(); err != nil {
			t.log.Warn("TCP Listener did not close cleanly", zap.Error(err))
		}
	}()

	// Listen for storage updates
	go func() {
		for update := range t.store.ListenToUpdates() {
			// TODO(rolly) deal with WriteUpdate error return
			t.WriteUpdate(update)
		}
	}()

	for {
		select {
		case <-t.ctx.Done():
			t.log.Info("Stopped accepting new connections")
			t.log.Info("Waiting for Read/Write loops to stop")
			loopWaiter.Wait()

			t.log.Info("Listener stopped")
			return nil

		default:
			conn, err := listener.Accept()
			if err != nil {
				netOpError := new(net.OpError)

				if errors.As(err, &netOpError) && netOpError.Unwrap().Error() == "use of closed network connection" {
					// The connection was closed while we were waiting for new connections
					// that's fine.
					return nil
				}

				// TODO(rolly) can we recover from some classes of err?
				return err
			}

			loopWaiter.Add(1)
			// writeQueue := make(chan []byte, 127)
			// t.writeQueues = append(t.writeQueues, writeQueue)
			tcpConn := NewTCPCOnn(t.ctx, conn.(*net.TCPConn), t.store, t.log.Named("conn"))

			t.addConn(tcpConn)

			go func() {
				defer loopWaiter.Done()
				tcpConn.Start()
			}()
		}
	}
}

func (t *TCPListener) WriteUpdate(update *storage.Update) (err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for conn := range t.activeConns {
		if uerr := conn.WriteUpdate(update); uerr != nil {
			err = multierr.Append(err, uerr)
		}
	}

	return err
}

func (t *TCPListener) addConn(conn *TCPConn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.activeConns[conn] = struct{}{}
}

func (t *TCPListener) removeConn(conn *TCPConn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.activeConns, conn)
}

type TCPConn struct {
	ctx        context.Context
	cancel     context.CancelFunc
	loopWaiter sync.WaitGroup

	conn  *net.TCPConn
	store storage.Store

	writeQueue chan []byte

	log *zap.Logger
}

func NewTCPCOnn(
	parentCtx context.Context,
	conn *net.TCPConn,
	store storage.Store,
	log *zap.Logger,
) *TCPConn {
	ctx, cancel := context.WithCancel(parentCtx)

	return &TCPConn{
		ctx:        ctx,
		cancel:     cancel,
		conn:       conn,
		store:      store,
		writeQueue: make(chan []byte, 127),
		log:        log,
	}
}

func (t *TCPConn) Close() error {
	if !t.isRunning() {
		// already stopped
		return nil
	}

	t.cancel()

	// Wait for the read/write loops to exit
	t.loopWaiter.Wait()

	t.conn.Close()

	// Once close is called, the writeQueue can no longer be used
	// We need to wait until the read/write loops have exited before
	// closing this channel.
	close(t.writeQueue)

	return nil
}

func (t *TCPConn) Start() {
	t.loopWaiter.Add(2)

	go func() {
		defer t.loopWaiter.Done()
		t.ReadLoop()
	}()

	go func() {
		defer t.loopWaiter.Done()
		t.WriteLoop()
	}()

	t.loopWaiter.Wait()
}

func (t *TCPConn) ReadLoop() {
	log := t.log.Named("readLoop")

	defer func() {
		log.Info("Listener read loop exiting")

		// Stop reading, but allow writes to drain
		err := t.conn.CloseRead()
		if err != nil && !strings.Contains(err.Error(), "transport endpoint is not connected") {
			log.Warn("Failed to close reads on connection cleanly",
				zap.Error(err))
		}

		log.Info("Listener read loop exited")
	}()

	for {
		select {
		case <-t.ctx.Done():
			log.Info("Context cancelled, exiting...")
			return

		default:
			// TODO(rolly) probably want to SetDeadline on the reads...
			req, err := protocol.ReadRequest(t.conn)
			if err != nil {
				log.Warn("Failed to read client request", zap.Error(err))
				continue
			}

			switch c := req.(type) {
			case *protocol.PingRequest:
				if err = protocol.WriteString(t, req.GetRequestID(), "PONG"); err != nil {
					log.Warn("Failed to respond to PING",
						zap.String("requestID", req.GetRequestID().String()))
					continue
				}

			case *protocol.QuitRequest:
				if err = protocol.WriteOk(t, req.GetRequestID()); err != nil {
					log.Warn("Failed to acknowledge QUICK",
						zap.String("requestID", req.GetRequestID().String()))
				}

				log.Info("Client QUIT, exiting...")

				return

			case *protocol.SetRequest:
				if err = t.dispatchSet(c); err != nil {
					log.Warn("Failed to dispatch set",
						zap.String("key", string(c.Key)),
						zap.String("requestID", req.GetRequestID().String()))
				}

			case *protocol.GetRequest:
				setCtx, cancel := context.WithTimeout(t.ctx, 3*time.Second)
				defer cancel()

				value, err := t.store.Get(setCtx, c.Key)

				if err != nil {
					log.Warn("Failed to get",
						zap.String("key", string(c.Key)),
						zap.Error(err))
					continue
				}

				if err := protocol.WriteLines(t, req.GetRequestID(), protocol.PrefixGet, value); err != nil {
					log.Warn("Failed to reply to get",
						zap.String("key", string(c.Key)),
						zap.String("value", string(value)),
						zap.Error(err))
					continue
				}
			}
		}
	}
}

func (t *TCPConn) WriteLoop() {
	log := t.log.Named("writeLoop")

	defer func() {
		log.Info("Listener write loop exiting")

		err := t.conn.CloseWrite()
		if err != nil && !strings.Contains(err.Error(), "transport endpoint is not connected") {
			log.Warn("Failed to close writes on connection cleanly",
				zap.Error(err))
		}

		log.Info("Listener write loop exited")
	}()

	// TODO(rolly) write the current store state on connect (i.e. now)

	for {
		select {
		case <-t.ctx.Done():
			return

		// These are responses from client requests handled by the read loop
		case data := <-t.writeQueue:
			t.log.Info("WRITE QUEUE", zap.String("data", string(data)))
			if data == nil {
				// Our ready loop has terminated, we should too
				log.Info("Write loop terminating as write queue has closed")
				return
			}

			if _, err := t.conn.Write(data); err != nil {
				t.log.Error("Failed to write from write queue",
					zap.String("data", string(data)),
					zap.Error(err))
				continue
			}
		}
	}
}

// Write writes data into the write for the write loop to write into the connection. Write! Write! Write!
func (t *TCPConn) Write(data []byte) (int, error) {
	if t.isRunning() {
		t.writeQueue <- data
	}

	return 0, nil
}

func (t *TCPConn) WriteUpdate(update *storage.Update) error {
	a := append(protocol.PrefixUpdate, update.Key...)
	a = append(a, '\n')

	buf := bytes.NewBuffer(a)

	if _, err := buf.Write(append(update.Value, '\n')); err != nil {
		return err
	}

	_, err := t.Write(buf.Bytes())
	return err
}

func (t *TCPConn) dispatchSet(req *protocol.SetRequest) error {
	setCtx, cancel := context.WithTimeout(t.ctx, 3*time.Second)
	defer cancel()

	if err := t.store.Set(setCtx, req.Key, req.Value); err != nil {
		return fmt.Errorf("Failed to set %w", err)
	}

	if err := protocol.WriteOk(t, req.GetRequestID()); err != nil {
		return fmt.Errorf("Failed to ack set %w", err)
	}

	return nil
}

// isRunning returns true if Close has not been called
func (t *TCPConn) isRunning() bool {
	select {
	case <-t.ctx.Done():
		// if we can read on this channel then it's been closed
		return false

	default:
		return true
	}
}
