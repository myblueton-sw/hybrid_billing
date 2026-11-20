package servicecatalog

import (
	"encoding/json"
	"unicode/utf8"
)

func hexUnit(data []byte, start int) (uint16, bool) {
	if start+4 > len(data) {
		return 0, false
	}
	var value uint16
	for _, c := range data[start : start+4] {
		value <<= 4
		switch {
		case c >= '0' && c <= '9':
			value += uint16(c - '0')
		case c >= 'a' && c <= 'f':
			value += uint16(c-'a') + 10
		case c >= 'A' && c <= 'F':
			value += uint16(c-'A') + 10
		default:
			return 0, false
		}
	}
	return value, true
}

func exactStrings(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	inside := false
	for i := 0; i < len(data); i++ {
		if !inside {
			inside = data[i] == '"'
			continue
		}
		if data[i] == '"' {
			inside = false
			continue
		}
		if data[i] != '\\' {
			continue
		}
		if i+1 == len(data) {
			return false
		}
		if data[i+1] != 'u' {
			i++
			continue
		}
		unit, valid := hexUnit(data, i+2)
		if !valid || unit >= 0xDC00 && unit <= 0xDFFF {
			return false
		}
		if unit >= 0xD800 && unit <= 0xDBFF {
			if i+8 > len(data) || data[i+6] != '\\' || data[i+7] != 'u' {
				return false
			}
			low, valid := hexUnit(data, i+8)
			if !valid || low < 0xDC00 || low > 0xDFFF {
				return false
			}
			i += 11
		} else {
			i += 5
		}
	}
	return !inside
}

func selectorWire(raw json.RawMessage) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || len(fields) != 1 {
		return false
	}
	if value, exists := fields["any"]; exists {
		var any bool
		return json.Unmarshal(value, &any) == nil && any
	}
	var id string
	return json.Unmarshal(fields["id"], &id) == nil && identifier(id)
}

func periodWire(raw json.RawMessage) bool {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return false
	}
	if value, exists := fields["any"]; exists {
		var any bool
		return len(fields) == 1 && json.Unmarshal(value, &any) == nil && any
	}
	var from, until string
	return len(fields) == 2 && json.Unmarshal(fields["from"], &from) == nil && json.Unmarshal(fields["until"], &until) == nil && from != "" && until != ""
}

func exactVariants(data []byte) bool {
	var wire struct {
		Audiences []struct {
			Scopes []struct {
				Customer    json.RawMessage `json:"customer"`
				LegalEntity json.RawMessage `json:"legal_entity"`
				Contract    json.RawMessage `json:"contract"`
				Period      json.RawMessage `json:"period"`
			} `json:"scopes"`
		} `json:"audiences"`
	}
	if json.Unmarshal(data, &wire) != nil {
		return false
	}
	for _, audience := range wire.Audiences {
		for _, scope := range audience.Scopes {
			if !selectorWire(scope.Customer) || !selectorWire(scope.LegalEntity) || !selectorWire(scope.Contract) || !periodWire(scope.Period) {
				return false
			}
		}
	}
	return true
}
