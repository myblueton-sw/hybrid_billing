package server

import (
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	pickupValidity     = 10 * time.Minute
	installationGrants = 30 * 24 * time.Hour
	invitationValidity = 7 * 24 * time.Hour
	maxRationale       = 256
)

type person struct {
	login string
	email string
}

type invite struct {
	id       string
	verifier identity.Verifier
	expires  time.Time
}

type outstanding struct {
	purpose string
	expires time.Time
}

type pickup struct {
	url        string
	secret     string
	credential string
	session    identity.Verifier
	target     string
	expires    time.Time
}

type directory struct {
	people      map[string]person
	taken       map[string]bool
	holders     map[string]time.Time
	invites     map[string]invite
	links       map[string]outstanding
	generations map[string]uint64
	pickups     map[string]pickup
}

func newDirectory() *directory {
	return &directory{people: map[string]person{}, taken: map[string]bool{}, holders: map[string]time.Time{}, invites: map[string]invite{},
		links: map[string]outstanding{}, generations: map[string]uint64{}, pickups: map[string]pickup{}}
}

func (d *directory) add(user, login, email string) {
	d.people[user] = person{login: login, email: email}
	d.taken[login] = true
}

func (d *directory) loginOf(user string) (string, bool) {
	found, ok := d.people[user]
	return found.login, ok
}

func (d *directory) userOf(login string) (string, bool) {
	for user, p := range d.people {
		if p.login == login {
			return user, true
		}
	}
	return "", false
}

func (d *directory) invitation(login string, now time.Time) (invite, bool) {
	found, ok := d.invites[login]
	return found, ok && now.Before(found.expires)
}

func (d *directory) link(login string, now time.Time) (outstanding, bool) {
	found, ok := d.links[login]
	return found, ok && now.Before(found.expires)
}

func (d *directory) liveInvites(now time.Time) int {
	count := 0
	for login := range d.invites {
		if _, ok := d.invitation(login, now); ok {
			count++
		}
	}
	return count
}

func (d *directory) liveLinks(now time.Time) int {
	count := 0
	for login := range d.links {
		if _, ok := d.link(login, now); ok {
			count++
		}
	}
	return count
}

func (d *directory) holds(user string, now time.Time) bool {
	expires, ok := d.holders[user]
	return ok && now.Before(expires)
}

func (d *directory) prune(now time.Time) {
	for ref, found := range d.pickups {
		if !now.Before(found.expires) {
			delete(d.pickups, ref)
		}
	}
	for login, found := range d.links {
		if !now.Before(found.expires) {
			delete(d.links, login)
		}
	}
}

func (d *directory) dropPickups(match func(pickup) bool) {
	for ref, found := range d.pickups {
		if match(found) {
			delete(d.pickups, ref)
		}
	}
}
