package transport

import "syscall"

type Poller struct {
	fd     int
	wakeFd int
}

func MakePoller() (*Poller, error) {
	var (
		poller Poller
		err    error
	)

	// Open an epoll fd
	// https://man7.org/linux/man-pages/man2/epoll_create.2.html
	poller.fd, err = syscall.EpollCreate1(0)

	if err != nil {
		return nil, err
	}

	// https://man7.org/linux/man-pages/man2/eventfd.2.html
	r0, _, e0 := syscall.Syscall(syscall.SYS_EVENTFD2, 0, 0, 0)
	if e0 != 0 {
		syscall.Close(poller.fd)
		panic(err)
	}
	poller.wakeFd = int(r0)

	// Register our interested for read and writes on our wakeFd
	// https://man7.org/linux/man-pages/man2/epoll_ctl.2.html
	event := &syscall.EpollEvent{Fd: int32(poller.wakeFd),
		Events: syscall.EPOLLIN | syscall.EPOLLOUT,
	}

	err = syscall.EpollCtl(poller.fd, syscall.EPOLL_CTL_ADD, poller.wakeFd, event)
	if err != nil {
		panic(err)
	}

	return &poller, nil
}

func (p *Poller) Close() error {
	if err := syscall.Close(p.wakeFd); err != nil {
		return err
	}

	return syscall.Close(p.fd)
}
