package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func sessionObject(reader io.Reader, allowed ...string) (map[string]json.RawMessage, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, 8193))
	if err != nil {
		return nil, err
	}
	if len(raw) > 8192 {
		return nil, &http.MaxBytesError{Limit: 8192}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errShape
	}
	out := map[string]json.RawMessage{}
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return nil, errShape
		}
		name, ok := token.(string)
		if !ok {
			return nil, errShape
		}
		known := false
		for _, value := range allowed {
			known = known || value == name
		}
		if !known || out[name] != nil {
			return nil, errShape
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return nil, errShape
		}
		out[name] = value
	}
	if token, err = decoder.Token(); err != nil || token != json.Delim('}') {
		return nil, errShape
	}
	if _, err = decoder.Token(); err != io.EOF {
		return nil, errShape
	}
	return out, nil
}
