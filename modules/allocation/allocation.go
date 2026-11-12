package allocation

import (
	"errors"
	"math/big"
	"sort"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

const (
	Algorithm               = "largest_remainder_v1"
	EncodingASCIIBytewiseV1 = "ascii_bytewise_v1"
	maxTargetIDBytes        = 128
)

var (
	ErrNoTargets         = errors.New("allocation has no targets")
	ErrTooManyTargets    = errors.New("allocation exceeds the target limit")
	ErrInvalidLimits     = errors.New("allocation limits are missing")
	ErrInvalidEncoding   = errors.New("target encoding is not supported")
	ErrInvalidTargetID   = errors.New("target identifier is outside the declared encoding")
	ErrDuplicateTargetID = errors.New("target identifier is repeated")
	ErrNegativeWeight    = errors.New("target weight is negative")
	ErrZeroTotalWeight   = errors.New("total weight is zero")
	ErrIntermediateBound = errors.New("allocation intermediate exceeds the declared limit")
	ErrConservation      = errors.New("allocation result does not reconcile")
)

type Limits struct {
	MaxTargets            int
	MaxIntermediateDigits int
}

type Target struct {
	ID     string
	Weight decimal.Decimal
}

type Share struct {
	ID     string
	Weight decimal.Decimal
	Amount decimal.Decimal
}

type Request struct {
	Profile  decimal.Profile
	Limits   Limits
	Encoding string
	Total    decimal.Decimal
	Quantum  decimal.Decimal
	Mode     decimal.Mode
	Targets  []Target
}

type Result struct {
	Algorithm         string
	Encoding          string
	ProfileID         string
	ProfileRevision   string
	Mode              decimal.Mode
	Quantum           decimal.Decimal
	Original          decimal.Decimal
	Quantized         decimal.Decimal
	RoundingComponent decimal.Decimal
	Denominator       decimal.Decimal
	Shares            []Share
}

func validID(id string) bool {
	if id == "" || len(id) > maxTargetIDBytes {
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

func digitCount(value *big.Int) int {
	if value.Sign() == 0 {
		return 1
	}
	return len(new(big.Int).Abs(value).Text(10))
}

func LargestRemainderV1(r Request) (Result, error) {
	if !r.Profile.Valid() {
		return Result{}, decimal.ErrInvalidProfile
	}
	if r.Limits.MaxTargets <= 0 || r.Limits.MaxIntermediateDigits <= 0 {
		return Result{}, ErrInvalidLimits
	}
	if r.Encoding != EncodingASCIIBytewiseV1 {
		return Result{}, ErrInvalidEncoding
	}
	if len(r.Targets) == 0 {
		return Result{}, ErrNoTargets
	}
	if len(r.Targets) > r.Limits.MaxTargets {
		return Result{}, ErrTooManyTargets
	}
	zero, err := decimal.New(r.Profile, new(big.Int), 0)
	if err != nil {
		return Result{}, err
	}
	seen := make(map[string]struct{}, len(r.Targets))
	denominator, widest := zero, 0
	for _, target := range r.Targets {
		if !validID(target.ID) {
			return Result{}, ErrInvalidTargetID
		}
		if _, repeated := seen[target.ID]; repeated {
			return Result{}, ErrDuplicateTargetID
		}
		seen[target.ID] = struct{}{}
		order, err := decimal.Cmp(r.Profile, target.Weight, zero)
		if err != nil {
			return Result{}, err
		}
		if order < 0 {
			return Result{}, ErrNegativeWeight
		}
		if denominator, err = decimal.Add(r.Profile, denominator, target.Weight); err != nil {
			return Result{}, err
		}
		widest = max(widest, target.Weight.Scale())
	}
	if denominator.Sign() == 0 {
		return Result{}, ErrZeroTotalWeight
	}
	quanta, err := decimal.Quanta(r.Profile, r.Total, r.Quantum, r.Mode)
	if err != nil {
		return Result{}, err
	}
	quantized, err := decimal.RoundToQuantum(r.Profile, r.Total, r.Quantum, r.Mode)
	if err != nil {
		return Result{}, err
	}
	component, err := decimal.Sub(r.Profile, r.Total, quantized)
	if err != nil {
		return Result{}, err
	}
	negative := quanta.Sign() < 0
	units := new(big.Int).Abs(quanta)
	weights := make([]*big.Int, len(r.Targets))
	sum := new(big.Int)
	for index, target := range r.Targets {
		if widest-target.Weight.Scale()+digitCount(target.Weight.Coefficient()) > r.Limits.MaxIntermediateDigits {
			return Result{}, ErrIntermediateBound
		}
		scaled := target.Weight.Coefficient()
		scaled.Mul(scaled, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(widest-target.Weight.Scale())), nil))
		weights[index] = scaled
		sum.Add(sum, scaled)
	}
	if digitCount(units)+digitCount(sum) > r.Limits.MaxIntermediateDigits {
		return Result{}, ErrIntermediateBound
	}
	counts := make([]*big.Int, len(r.Targets))
	remainders := make([]*big.Int, len(r.Targets))
	assigned := new(big.Int)
	for index, weight := range weights {
		counts[index], remainders[index] = new(big.Int).QuoRem(new(big.Int).Mul(units, weight), sum, new(big.Int))
		assigned.Add(assigned, counts[index])
	}
	order := make([]int, len(r.Targets))
	for index := range order {
		order[index] = index
	}
	sort.Slice(order, func(a, b int) bool {
		if c := remainders[order[a]].Cmp(remainders[order[b]]); c != 0 {
			return c > 0
		}
		return r.Targets[order[a]].ID < r.Targets[order[b]].ID
	})
	left := new(big.Int).Sub(units, assigned)
	if !left.IsInt64() || left.Sign() < 0 || left.Int64() > int64(len(order)) {
		return Result{}, ErrConservation
	}
	one := big.NewInt(1)
	for position := 0; position < int(left.Int64()); position++ {
		counts[order[position]].Add(counts[order[position]], one)
	}
	shares := make([]Share, len(r.Targets))
	total := zero
	for index, target := range r.Targets {
		count := counts[index]
		if negative {
			count.Neg(count)
		}
		amount, err := decimal.New(r.Profile, count.Mul(count, r.Quantum.Coefficient()), r.Quantum.Scale())
		if err != nil {
			return Result{}, err
		}
		if total, err = decimal.Add(r.Profile, total, amount); err != nil {
			return Result{}, err
		}
		shares[index] = Share{ID: target.ID, Weight: target.Weight, Amount: amount}
	}
	sort.Slice(shares, func(a, b int) bool { return shares[a].ID < shares[b].ID })
	restored, err := decimal.Add(r.Profile, total, component)
	if err != nil {
		return Result{}, err
	}
	if !total.Equal(quantized) || !restored.Equal(r.Total) {
		return Result{}, ErrConservation
	}
	return Result{
		Algorithm:         Algorithm,
		Encoding:          r.Encoding,
		ProfileID:         r.Profile.ID(),
		ProfileRevision:   r.Profile.Revision(),
		Mode:              r.Mode,
		Quantum:           r.Quantum,
		Original:          r.Total,
		Quantized:         quantized,
		RoundingComponent: component,
		Denominator:       denominator,
		Shares:            shares,
	}, nil
}
