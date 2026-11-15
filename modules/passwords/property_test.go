package passwords

import (
	"math/rand"
	"testing"
	"time"
)

type mAccount struct {
	password string
	failures int
	first    time.Time
	locked   time.Time
}

type mLink struct {
	key      Key
	expires  time.Time
	used     bool
	reserved time.Time
}

func TestProperty(t *testing.T) {
	keys := []Key{{Issuer: "i", Subject: "a@x"}, {Issuer: "i", Subject: "b@x"}}
	words := []string{"first password", "second password", "third password"}
	for seed := int64(1); seed <= 80; seed++ {
		rng := rand.New(rand.NewSource(seed))
		w := setup(t)
		accounts := map[Key]*mAccount{}
		links := map[string]*mLink{}
		current := map[Key]string{}
		holds := map[string]time.Time{}
		now := start
		live := func(tok string) bool {
			l := links[tok]
			return l != nil && !l.used && !now.Before(l.reserved) && now.Before(l.expires)
		}
		dropKey := func(k Key) {
			if tok, ok := current[k]; ok {
				delete(links, tok)
				delete(current, k)
			}
		}
		for step := 0; step < 60; step++ {
			k := keys[rng.Intn(2)]
			word := words[rng.Intn(3)]
			switch rng.Intn(8) {
			case 0:
				rec, _ := w.r.Hash(k, word)
				w.r.Put(k, rec)
				accounts[k] = &mAccount{password: word}
				dropKey(k)
			case 1, 2:
				got := w.r.Verify(k, word)
				want := ResultWrong
				if a := accounts[k]; a != nil {
					switch {
					case now.Before(a.locked):
						want = ResultLocked
					case a.password == word:
						want, a.failures = ResultOK, 0
					default:
						if a.failures == 0 || now.Sub(a.first) >= 15*time.Minute {
							a.failures, a.first = 0, now
						}
						a.failures++
						if a.failures >= 5 {
							a.locked, a.failures = now.Add(15*time.Minute), 0
						}
					}
				}
				if got != want {
					t.Fatalf("seed %d step %d verify %s want %s", seed, step, got, want)
				}
			case 3:
				d := time.Duration(rng.Intn(10)) * time.Minute
				w.advance(d)
				now = now.Add(d)
			case 4:
				tok, err := w.r.Issue(k, PurposeSet, time.Hour)
				if err != nil {
					t.Fatal(err)
				}
				dropKey(k)
				links[tok], current[k] = &mLink{key: k, expires: now.Add(time.Hour)}, tok
			case 5:
				tok, ok := current[k]
				if !ok {
					continue
				}
				l, err := w.r.Reserve(tok)
				if live(tok) != (err == nil) {
					t.Fatalf("seed %d reserve", seed)
				}
				if err == nil {
					links[tok].reserved = now.Add(5 * time.Minute)
					holds[tok] = l.Hold
				}
			case 6:
				tok, ok := current[k]
				if !ok {
					continue
				}
				rec, _ := w.r.Hash(k, word)
				_, err := w.r.Commit(tok, rec)
				l := links[tok]
				committed := !l.used && now.Before(l.reserved)
				if committed != (err == nil) {
					t.Fatalf("seed %d step %d commit %v", seed, step, err)
				}
				if committed {
					accounts[k] = &mAccount{password: word}
					dropKey(k)
				}
			case 7:
				tok, ok := current[k]
				if !ok {
					continue
				}
				w.r.Release(tok, holds[tok])
				if l := links[tok]; !l.used && now.Before(l.reserved) {
					l.reserved = time.Time{}
				}
			}
			for tok := range links {
				if _, err := w.r.Peek(tok); (err == nil) != live(tok) {
					t.Fatalf("seed %d step %d peek", seed, step)
				}
			}
		}
	}
}
