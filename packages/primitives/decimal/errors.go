package decimal

import "errors"

var (
	ErrSyntax            = errors.New("decimal text is not a finite base-10 number")
	ErrInputBound        = errors.New("decimal input exceeds the profile input bounds")
	ErrResultBound       = errors.New("decimal result exceeds the profile result bounds")
	ErrIntermediateBound = errors.New("decimal intermediate exceeds the profile intermediate bound")
	ErrInexactDivision   = errors.New("division has no exact result within the profile bounds")
	ErrDivisionByZero    = errors.New("division by zero")
	ErrInvalidMode       = errors.New("rounding mode is not supported")
	ErrInvalidProfile    = errors.New("numeric profile is missing or incomplete")
	ErrInvalidQuantum    = errors.New("quantum must be positive")
	ErrInvalidScale      = errors.New("scale is outside the profile result scale")
	ErrInvalidOperand    = errors.New("decimal operand is missing")
	ErrInvalidCurrency   = errors.New("currency code is not three uppercase letters")
	ErrCurrencyMismatch  = errors.New("currencies differ")
)
