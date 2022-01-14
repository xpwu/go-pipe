package pipe

import (
  "errors"
  "strings"
)

const Network = "pipe"

type addr struct {
  str string
}

func newAddr(str string) (a *addr, err error) {
  if !strings.HasPrefix(str, Network+":") {
   err = errors.New("must be 'pipe:xxx'")
   return
  }

  return &addr{str:str}, nil
}

func (a *addr) Network() string {
  return Network
}

func (a *addr) String() string {
  return a.str
}



