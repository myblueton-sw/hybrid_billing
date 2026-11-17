package passwords

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"sync"
	"time"
	"unicode/utf8"
)

type account struct {
	record   Record
	failures int
	first    time.Time
	locked   time.Time
}

type link struct {
	view     Link
	used     bool
	reserved time.Time
}

type Registry struct {
	mu       sync.Mutex
	clock    func() time.Time
	params   Params
	accounts map[Key]*account
	links    map[[32]byte]*link
	byKey    map[Key][32]byte
	dummy    Record
	between  func()
}

func (p Params) valid() bool {
	return p.Iterations >= 1 && p.SaltBytes >= 16 && p.KeyBytes >= 16 && p.MinLength >= 1 && p.MaxLength >= p.MinLength && p.Attempts >= 1 &&
		p.Lockout > 0 && p.MinValidity > 0 && p.MaxValidity >= p.MinValidity && p.Records >= 1 && p.Links >= 1
}

func validKey(k Key) bool { return validID(k.Issuer) && validText(k.Subject) }

func New(clock func() time.Time, params Params) (*Registry, error) {
	if clock == nil || !params.valid() {
		return nil, ErrInvalid
	}
	r := &Registry{clock: clock, params: params, accounts: map[Key]*account{}, links: map[[32]byte]*link{}, byKey: map[Key][32]byte{}}
	r.dummy = r.derive(make([]byte, params.SaltBytes), []byte("dummy password for timing"))
	return r, nil
}

func (r *Registry) derive(salt, password []byte) Record {
	key, _ := pbkdf2.Key(sha256.New, string(password), salt, r.params.Iterations, r.params.KeyBytes)
	return Record{Iterations: r.params.Iterations, Salt: append([]byte{}, salt...), Hash: key}
}

func (r *Registry) form(k Key, password string) error {
	switch {
	case !validKey(k):
		return ErrInvalid
	case !utf8.ValidString(password) || len(password) < r.params.MinLength || len(password) > r.params.MaxLength:
		return Refusal{Reason: ReasonPasswordForm}
	case password == k.Subject:
		return Refusal{Reason: ReasonSameAsLogin}
	}
	return nil
}

func (r *Registry) Hash(k Key, password string) (Record, error) {
	if err := r.form(k, password); err != nil {
		return Record{}, err
	}
	salt := make([]byte, r.params.SaltBytes)
	_, _ = rand.Read(salt)
	return r.derive(salt, []byte(password)), nil
}

func (r *Registry) Put(k Key, record Record) error {
	if !validKey(k) || !r.validRecord(record) {
		return ErrInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.accounts[k] == nil && len(r.accounts) >= r.params.Records {
		return ErrLimit
	}
	r.store(k, record)
	r.dropLink(k)
	return nil
}

func (r *Registry) store(k Key, record Record) {
	r.accounts[k] = &account{record: Record{Iterations: record.Iterations, Salt: append([]byte{}, record.Salt...), Hash: append([]byte{}, record.Hash...)}}
}

func (r *Registry) validRecord(record Record) bool {
	return record.Iterations == r.params.Iterations && len(record.Salt) == r.params.SaltBytes && len(record.Hash) == r.params.KeyBytes
}

func (r *Registry) Remove(k Key) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.accounts, k)
	r.dropLink(k)
}

func (r *Registry) Has(k Key) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.accounts[k] != nil
}

func (r *Registry) compare(k Key, password string) (*account, bool) {
	r.mu.Lock()
	found := r.accounts[k]
	record := r.dummy
	if found != nil {
		record = found.record
	}
	r.mu.Unlock()
	if r.between != nil {
		r.between()
	}
	got := r.derive(record.Salt, []byte(password))
	return found, subtle.ConstantTimeCompare(got.Hash, record.Hash) == 1
}

func (r *Registry) VerifyHolder(k Key, password string) string {
	found, match := r.compare(k, password)
	r.mu.Lock()
	defer r.mu.Unlock()
	if found == nil || r.accounts[k] != found || !match {
		return ResultWrong
	}
	return ResultOK
}

func (r *Registry) Verify(k Key, password string) string {
	now := r.clock()
	found, match := r.compare(k, password)
	if found == nil {
		return ResultWrong
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch {
	case r.accounts[k] != found:
		return ResultWrong
	case now.Before(found.locked):
		return ResultLocked
	case match:
		found.failures = 0
		return ResultOK
	}
	if found.failures == 0 || now.Sub(found.first) >= r.params.Lockout {
		found.failures, found.first = 0, now
	}
	found.failures++
	if found.failures >= r.params.Attempts {
		found.locked = now.Add(r.params.Lockout)
	}
	return ResultWrong
}
