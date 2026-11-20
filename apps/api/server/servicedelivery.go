package server

import "math"

func (d *directory) deliveryChanged() {
	if d.deliveryVersion < math.MaxUint64 {
		d.deliveryVersion++
	}
}

func (d *directory) dropPickup(ref string) {
	found, ok := d.pickups[ref]
	if !ok {
		return
	}
	if found.credential != "" && d.pickupByCredential[found.credential] == ref {
		delete(d.pickupByCredential, found.credential)
	}
	delete(d.pickups, ref)
	d.deliveryChanged()
}

func (d *directory) putPickup(ref string, found pickup) {
	d.dropPickup(ref)
	if found.credential != "" {
		if prior := d.pickupByCredential[found.credential]; prior != "" {
			d.dropPickup(prior)
		}
		d.pickupByCredential[found.credential] = ref
	}
	d.pickups[ref] = found
	d.deliveryChanged()
}

func (d *directory) credentialPickup(id string) (pickup, bool) {
	ref, ok := d.pickupByCredential[id]
	if !ok {
		return pickup{}, false
	}
	found, ok := d.pickups[ref]
	return found, ok && found.credential == id
}

func (d *directory) putDelivery(id string, found delivery) {
	d.collected[id] = found
	d.deliveryChanged()
}

func (d *directory) dropDelivery(id string) {
	if _, ok := d.collected[id]; ok {
		delete(d.collected, id)
		d.deliveryChanged()
	}
}
