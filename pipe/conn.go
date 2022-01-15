package pipe

import (
  "net"
  "time"
)

type conn struct {
  reader *Reader
  writer *Writer
  addr *addr
}

func connPair(addr *addr) (pair [2]*conn) {
  pair[0] = &conn{addr:addr}
  pair[1] = &conn{addr:addr}

  pair[0].reader, pair[1].writer = Pipe()
  pair[1].reader, pair[0].writer = Pipe()

  return
}

func (c *conn) Read(b []byte) (n int, err error) {
  return c.reader.Read(b)
}

func (c *conn) Write(b []byte) (n int, err error) {
  return c.writer.Write(b)
}

func (c *conn) Close() error {
  err := c.writer.Close()
  if err != nil {
    return err
  }

  err = c.reader.Close()
  if err != nil {
    return err
  }

  return nil
}

func (c *conn) LocalAddr() net.Addr {
  return c.addr
}

func (c *conn) RemoteAddr() net.Addr {
  return c.addr
}

func (c *conn) SetDeadline(t time.Time) error {
  c.reader.SetDeadline(t)
  c.writer.SetDeadline(t)
  return nil
}

func (c *conn) SetReadDeadline(t time.Time) error {
  c.reader.SetDeadline(t)
  return nil
}

func (c *conn) SetWriteDeadline(t time.Time) error {
  c.writer.SetDeadline(t)
  return nil
}
