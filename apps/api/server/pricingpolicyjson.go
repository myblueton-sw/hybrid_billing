package server

import (
	"encoding/json"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/rating"
)

const pricingPolicyFieldBytes = 4096

func pricingJSONUnicode(raw string) bool {
	if !utf8.ValidString(raw) {
		return false
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] != '\\' {
			continue
		}
		if i+1 >= len(raw) {
			return false
		}
		if raw[i+1] != 'u' {
			i++
			continue
		}
		if i+6 > len(raw) {
			return false
		}
		unit, err := strconv.ParseUint(raw[i+2:i+6], 16, 16)
		if err != nil || unit >= 0xdc00 && unit <= 0xdfff {
			return false
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			if i+12 > len(raw) || raw[i+6:i+8] != "\\u" {
				return false
			}
			low, err := strconv.ParseUint(raw[i+8:i+12], 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 11
		} else {
			i += 5
		}
	}
	return true
}

func pricingJSONShape(d *json.Decoder, typ reflect.Type, depth int) bool {
	if depth > 12 {
		return false
	}
	token, err := d.Token()
	if err != nil || token == nil {
		return false
	}
	if typ == reflect.TypeOf(time.Time{}) {
		_, ok := token.(string)
		return ok
	}
	switch typ.Kind() {
	case reflect.Struct:
		if token != json.Delim('{') {
			return false
		}
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			name, ok := key.(string)
			field, found := typ.FieldByName(name)
			if err != nil || !ok || !found || seen[name] || field.PkgPath != "" || !pricingJSONShape(d, field.Type, depth+1) {
				return false
			}
			seen[name] = true
		}
		end, err := d.Token()
		return err == nil && end == json.Delim('}')
	case reflect.Slice:
		if token != json.Delim('[') {
			return false
		}
		for d.More() {
			if !pricingJSONShape(d, typ.Elem(), depth+1) {
				return false
			}
		}
		end, err := d.Token()
		return err == nil && end == json.Delim(']')
	case reflect.String:
		_, ok := token.(string)
		return ok
	case reflect.Int:
		v, ok := token.(json.Number)
		if !ok {
			return false
		}
		_, err := strconv.ParseInt(string(v), 10, typ.Bits())
		return err == nil
	}
	return false
}

func pricingSpecJSON(spec pricingpolicy.Spec) (string, error) {
	spec.Policy.Eligible = append([]rating.Class{}, spec.Policy.Eligible...)
	spec.Policy.Rules = append([]rating.Rule{}, spec.Policy.Rules...)
	for i := range spec.Policy.Rules {
		r := &spec.Policy.Rules[i]
		r.Tiers = append([]rating.Tier{}, r.Tiers...)
		r.Discounts = append([]rating.Discount{}, r.Discounts...)
	}
	body, err := json.Marshal(spec)
	if err != nil || len(body) > pricingPolicyFieldBytes {
		return "", pricingpolicy.ErrInvalid
	}
	return string(body), nil
}

func parsePricingSpec(raw, workspace string) (pricingpolicy.Spec, error) {
	var spec pricingpolicy.Spec
	if len(raw) == 0 || len(raw) > pricingPolicyFieldBytes || !pricingJSONUnicode(raw) {
		return spec, pricingpolicy.ErrInvalid
	}
	d := json.NewDecoder(strings.NewReader(raw))
	d.UseNumber()
	if !pricingJSONShape(d, reflect.TypeOf(spec), 0) {
		return spec, pricingpolicy.ErrInvalid
	}
	if _, err := d.Token(); err != io.EOF {
		return spec, pricingpolicy.ErrInvalid
	}
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		return spec, pricingpolicy.ErrInvalid
	}
	if err := pricingpolicy.ValidateSpec(spec); err != nil {
		return spec, err
	}
	if spec.Scope.Workspace != workspace {
		return spec, pricingpolicy.ErrRejected
	}
	if _, err := pricingSpecJSON(spec); err != nil {
		return spec, err
	}
	return spec, nil
}
