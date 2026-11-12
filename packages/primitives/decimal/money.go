package decimal

type Money struct {
	amount   Decimal
	currency string
}

func NewMoney(amount Decimal, currency string) (Money, error) {
	if !amount.Valid() {
		return Money{}, ErrInvalidOperand
	}
	if len(currency) != 3 {
		return Money{}, ErrInvalidCurrency
	}
	for index := 0; index < 3; index++ {
		if currency[index] < 'A' || currency[index] > 'Z' {
			return Money{}, ErrInvalidCurrency
		}
	}
	return Money{amount: amount, currency: currency}, nil
}

func (m Money) Amount() Decimal { return m.amount }

func (m Money) Currency() string { return m.currency }

func combine(p Profile, a, b Money, operation func(Profile, Decimal, Decimal) (Decimal, error)) (Money, error) {
	if !a.amount.Valid() || !b.amount.Valid() {
		return Money{}, ErrInvalidOperand
	}
	if a.currency != b.currency {
		return Money{}, ErrCurrencyMismatch
	}
	amount, err := operation(p, a.amount, b.amount)
	if err != nil {
		return Money{}, err
	}
	return Money{amount: amount, currency: a.currency}, nil
}

func AddMoney(p Profile, a, b Money) (Money, error) { return combine(p, a, b, Add) }

func SubMoney(p Profile, a, b Money) (Money, error) { return combine(p, a, b, Sub) }
