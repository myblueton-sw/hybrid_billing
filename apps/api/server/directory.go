package server

import (
	"sort"
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

type delivery struct {
	service   string
	collected bool
}

type directory struct {
	people      map[string]person
	byLogin     map[string]string
	order       []string
	departed    []string
	version     uint64
	taken       map[string]bool
	holders     map[string]time.Time
	invites     map[string]invite
	links       map[string]outstanding
	generations map[string]uint64
	pickups     map[string]pickup
	collected   map[string]delivery
}

func newDirectory() *directory {
	return &directory{people: map[string]person{}, byLogin: map[string]string{}, taken: map[string]bool{}, holders: map[string]time.Time{}, invites: map[string]invite{},
		links: map[string]outstanding{}, generations: map[string]uint64{}, pickups: map[string]pickup{}, collected: map[string]delivery{}}
}

func (d *directory) position(user, login string) int {
	return sort.Search(len(d.order), func(i int) bool {
		other := d.order[i]
		at := d.people[other].login
		return at > login || at == login && other >= user
	})
}

func (d *directory) add(user, login, email string) {
	d.remove(user)
	at := d.position(user, login)
	d.people[user] = person{login: login, email: email}
	d.byLogin[login] = user
	d.taken[login] = true
	d.order = append(d.order, "")
	copy(d.order[at+1:], d.order[at:])
	d.order[at] = user
	d.version++
}

func (d *directory) remove(user string) {
	found, ok := d.people[user]
	if !ok {
		return
	}
	at := d.position(user, found.login)
	d.order = append(d.order[:at], d.order[at+1:]...)
	delete(d.people, user)
	delete(d.byLogin, found.login)
	d.version++
}

func (d *directory) depart(user string) {
	d.remove(user)
	at := sort.SearchStrings(d.departed, user)
	if at < len(d.departed) && d.departed[at] == user {
		return
	}
	d.departed = append(d.departed, "")
	copy(d.departed[at+1:], d.departed[at:])
	d.departed[at] = user
	d.version++
}

func (d *directory) invite(login string, found invite) {
	d.invites[login] = found
	d.version++
}

func (d *directory) dropInvite(login string) {
	if _, ok := d.invites[login]; ok {
		delete(d.invites, login)
		d.version++
	}
}

func (d *directory) issued(login string, found outstanding) {
	d.links[login] = found
	d.version++
}

func (d *directory) dropLink(login string) {
	if _, ok := d.links[login]; ok {
		delete(d.links, login)
		d.version++
	}
}

func (d *directory) grant(user string, expires time.Time) {
	d.holders[user] = expires
	d.version++
}

func (d *directory) dropHolder(user string) {
	if _, ok := d.holders[user]; ok {
		delete(d.holders, user)
		d.version++
	}
}

func (d *directory) loginOf(user string) (string, bool) {
	found, ok := d.people[user]
	return found.login, ok
}

func (d *directory) userOf(login string) (string, bool) {
	user, ok := d.byLogin[login]
	return user, ok
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

func (d *directory) dropPickupRefs(refs []string) {
	for _, ref := range refs {
		delete(d.pickups, ref)
	}
}

func (d *directory) forget(service string, keep map[string]bool) {
	for id, found := range d.collected {
		if found.service == service && !keep[id] {
			delete(d.collected, id)
		}
	}
}
