package server

import (
	"bytes"

	"github.com/myblueton-sw/hybrid_billing/modules/servicecatalog"
)

func (s *Server) configureServiceCatalog(input []byte) error {
	var err error
	if input == nil {
		s.serviceCatalog, err = servicecatalog.New(servicecatalog.Config{}, s.gate)
	} else {
		s.serviceCatalog, err = servicecatalog.Decode(bytes.NewReader(input), s.gate)
	}
	if err != nil {
		return ErrConfig
	}
	return nil
}
