package pipe

import (
  "context"
  "github.com/stretchr/testify/assert"
  "net"
  "testing"
)

func must(d interface{}, err error) interface{} {
  if err != nil {
    panic(err)
  }

  return d
}

func TestPipe(t *testing.T) {
  addr := "pipe:qkjjjdd"
  t.Log(addr)

  l := must(Listen(addr)).(net.Listener)
  go func() {
    for {
      c := must(l.Accept()).(net.Conn)

      go func() {
        b := make([]byte, 100)

        for {
          n := must(c.Read(b)).(int)
          b = b[:n]
          _, _ = c.Write(b)
        }
      }()
    }

  }()

  client := must(Dial(context.Background(), addr)).(net.Conn)
  s := "this is test"
  _ = must(client.Write([]byte(s))).(int)

  r := make([]byte, 100)
  n := must(client.Read(r)).(int)

  a := assert.New(t)
  a.EqualValues([]byte(s), r[:n])
}
