package servicecatalog

import (
	"bytes"
	"encoding/json"
	"io"
)

var wireFields = map[string]bool{
	"audiences": true, "workspace": true, "id": true, "resource": true, "scopes": true,
	"action": true, "kind": true, "fields": true, "customer": true, "legal_entity": true,
	"contract": true, "period": true, "any": true, "from": true, "until": true,
}

func uniqueValue(decoder *json.Decoder, depth int) error {
	if depth > 64 {
		return ErrInvalid
	}
	token, err := decoder.Token()
	if err != nil || token == nil {
		return ErrInvalid
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			key, err := decoder.Token()
			name, valid := key.(string)
			if err != nil || !valid || seen[name] || !wireFields[name] {
				return ErrInvalid
			}
			seen[name] = true
			if err := uniqueValue(decoder, depth+1); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return ErrInvalid
		}
	case '[':
		for decoder.More() {
			if err := uniqueValue(decoder, depth+1); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}

func Decode(reader io.Reader, validator Validator) (*Catalog, error) {
	if reader == nil {
		return nil, ErrInvalid
	}
	data, err := io.ReadAll(io.LimitReader(reader, MaxBytes+1))
	if err != nil || len(data) > MaxBytes || !exactStrings(data) {
		return nil, ErrInvalid
	}
	unique := json.NewDecoder(bytes.NewReader(data))
	unique.UseNumber()
	if err := uniqueValue(unique, 0); err != nil {
		return nil, ErrInvalid
	}
	if _, err := unique.Token(); err != io.EOF {
		return nil, ErrInvalid
	}
	var wire struct {
		Audiences *[]AudienceSpec `json:"audiences"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil || wire.Audiences == nil {
		return nil, ErrInvalid
	}
	for _, audience := range *wire.Audiences {
		if audience.Scopes == nil {
			return nil, ErrInvalid
		}
	}
	if !exactVariants(data) {
		return nil, ErrInvalid
	}
	return New(Config{Audiences: *wire.Audiences}, validator)
}
