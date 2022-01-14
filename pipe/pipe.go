package pipe

import (
  "context"
  "net"
)

// addr: <unionFlag>

func Dial(ctx context.Context, addr string) (c net.Conn, err error) {
  l,err := allListeners.get(addr)
  if err != nil {
    return
  }

  return l.connect(ctx)
}

func Listen(addr string) (l net.Listener, err error) {
  a,err := newAddr(addr)
  if err != nil {
    return
  }

  ls := newListener(a)

  if err = allListeners.add(addr, ls); err != nil {
    return
  }

  l = ls
  return
}



