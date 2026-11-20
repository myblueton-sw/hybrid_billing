package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

type serverInputAuthority = serverGrantAuthority

func newServerInputAuthority(s *Server, limits inputauthority.Limits) (*serverInputAuthority, error) {
	return newServerGrantAuthority(s, limits, inputGrantAuthority)
}
