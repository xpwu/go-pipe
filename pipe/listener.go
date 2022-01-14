package pipe

import (
  "context"
  "errors"
  "net"
  "sync"
)

type listener struct {
  addr   *addr
  accept chan net.Conn
  closed bool
}

func newListener(a *addr) *listener {
  return &listener{addr: a, accept: make(chan net.Conn)}
}

func (l *listener) Accept() (net.Conn, error) {
  c := <-l.accept

  if c == nil {
    return nil, errors.New("closed")
  }
  return c, nil
}

func (l *listener) Close() error {
  if l.closed {
    return errors.New("has been closed")
  }
  l.closed = true

  l.accept <- nil
  return nil
}

func (l *listener) Addr() net.Addr {
  return l.addr
}

func (l *listener) connect(ctx context.Context) (net.Conn, error) {
  conns := connPair(l.addr)
  select {
  case l.accept <- conns[0]:
    // nothing to do
  case <-ctx.Done():
    return nil, ctx.Err()
  }

  return conns[1], nil
}


// todo: 修改为 chan 实现
var (
  allListeners = &listeners{values: make(map[string]*listener)}
)

type listeners struct {
  values map[string]*listener
  mu     sync.RWMutex
}

func (ls *listeners) add(addr string, l *listener) error {
  ls.mu.Lock()
  defer ls.mu.Unlock()

  if _, ok := ls.values[addr]; ok {
    return errors.New("listen " + addr + " duplicated")
  }

  ls.values[addr] = l
  return nil
}

func (ls *listeners) get(addr string) (l *listener, err error) {
  ls.mu.RLock()
  defer ls.mu.RUnlock()

  l, ok := ls.values[addr]
  if !ok {
    err = errors.New("connect " + addr + " error. NOT be bind")
  }

  return
}
