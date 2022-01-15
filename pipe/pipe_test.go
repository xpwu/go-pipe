package pipe

import (
  "context"
  "github.com/stretchr/testify/assert"
  "net"
  "os"
  "testing"
  "time"
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

func TestTimeout(t *testing.T) {
  addr := "pipe:timeout"
  t.Log(addr)
  a := assert.New(t)

  l := must(Listen(addr)).(net.Listener)
  go func() {
    for {
      c := must(l.Accept()).(net.Conn)

      go func() {
       for {
         // 接收一个数据，写数据方也会出现timeout的情况
         b := make([]byte, 1)
         _ = c.SetDeadline(time.Now().Add(1500 * time.Millisecond))
         must(c.Read(b))
         a.Equal("t", string(b))

         b = append(b, 0x30)
         // 客户端没有读取，这里应该是要超时的
         _, err := c.Write(b)
         a.Equal(os.ErrDeadlineExceeded, err)
         if err != nil {
          break
         }
       }
      }()
    }

  }()

  client := must(Dial(context.Background(), addr)).(net.Conn)
  _ = client.SetWriteDeadline(time.Now().Add(1 * time.Second))

  s := "this is test"
  _, err := client.Write([]byte(s))
  a.Equal(os.ErrDeadlineExceeded, err)

  // 等待服务器写超时，再测试客户端的读超时
  time.Sleep(2 * time.Second)

  r := make([]byte, 0)
  _ = client.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
  n, err := client.Read(r)
  a.Equal(os.ErrDeadlineExceeded, err)

  a.NotEqual([]byte(s), r[:n])
}
