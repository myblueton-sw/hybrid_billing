package cases

import "time"

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func optionalID(id string) bool { return id == "" || validID(id) }

func validCurrency(code string) bool {
	return len(code) == 3 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' && code[2] >= 'A' && code[2] <= 'Z'
}

func validDate(text string) bool {
	if len(text) != 10 || text < "2000-01-01" || text > "2099-12-31" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", text)
	return err == nil && parsed.Format("2006-01-02") == text
}

func validDigest(text string) bool {
	if len(text) != 64 {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func validNumber(text string, signed bool) bool {
	if text == "" || len(text) > maxNumberBytes {
		return false
	}
	body := text
	if body[0] == '-' {
		if !signed {
			return false
		}
		body = body[1:]
	}
	whole, fraction := body, ""
	for index := 0; index < len(body); index++ {
		if body[index] == '.' {
			whole, fraction = body[:index], body[index+1:]
			if fraction == "" || len(fraction) > maxFraction {
				return false
			}
			break
		}
	}
	if whole == "" || len(whole) > 1 && whole[0] == '0' || !digits(whole) || !digits(fraction) {
		return false
	}
	return !(text[0] == '-' && zero(whole, fraction))
}

func digits(text string) bool {
	for index := 0; index < len(text); index++ {
		if text[index] < '0' || text[index] > '9' {
			return false
		}
	}
	return true
}

func zero(whole, fraction string) bool {
	for _, part := range []string{whole, fraction} {
		for index := 0; index < len(part); index++ {
			if part[index] != '0' {
				return false
			}
		}
	}
	return true
}

func validActor(a Actor) bool {
	if !validID(a.ID) || !sides[a.Side] {
		return false
	}
	if a.Side == SideCustomer {
		return validID(a.Customer)
	}
	return a.Customer == ""
}
