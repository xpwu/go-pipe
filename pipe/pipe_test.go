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

  l := must(Listen(addr)).(net.Listener)
  go func() {
    c := must(l.Accept()).(net.Conn)
    b := make([]byte, 0, 100)

    go func() {
      _, _ = c.Read(b)
      print(b)
      _, _ = c.Write(b)
    }()

  }()

  client := must(Dial(context.Background(), addr)).(net.Conn)
  s := "this is test"
  _, _ = client.Write([]byte(s))
  r := make([]byte, 0, 100)
  _, _ = client.Read(r)

  a := assert.New(t)
  a.EqualValues(s, r)
}
