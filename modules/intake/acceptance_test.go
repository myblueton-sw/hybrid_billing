package intake

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestInterruptedReceiptOfManyRecordsKeepsCountSumsAndDigest(t *testing.T) {
	const chunkCount, perChunk = 120, 1000
	random := rand.New(rand.NewSource(186))
	currencies := []string{"USD", "EUR", "KRW"}
	chunks := make([][]Record, chunkCount)
	sums := map[string]int64{}
	var every []Record
	for number := 0; number < chunkCount; number++ {
		for index := 0; index < perChunk; index++ {
			sequence := number*perChunk + index
			var value int64
			currency := currencies[random.Intn(len(currencies))]
			switch {
			case sequence%10 == 8:
				currency, value = "ZRO", int64(random.Intn(1000000))+1
			case sequence%10 == 9:
				currency, value = "ZRO", -scaled(t, every[sequence-1].Amount)
			default:
				value = int64(random.Intn(2000000000)) - 400000000
			}
			sums[currency] += value
			record := rec(fmt.Sprintf("evt-%07d", (sequence*7919)%(chunkCount*perChunk)), unscaled(value), currency)
			chunks[number] = append(chunks[number], record)
			every = append(every, record)
		}
	}
	if sums["ZRO"] != 0 || len(every) != 120000 {
		t.Fatal("generator")
	}
	declared := map[string]string{}
	for currency, sum := range sums {
		declared[currency] = unscaled(sum)
	}
	spec := base("a1", "d1", 1)
	spec.Holds, spec.Chunks, spec.Rows, spec.Sums, spec.Totals = Rows, chunkCount, len(every), declared, SourceStated
	change := func(l *Limits) { l.Page = 5000 }

	straight := newHarness(t, change)
	straight.register(spec)
	straight.stage(spec, chunks...)

	broken := newHarness(t, change)
	broken.register(spec)
	send := func(number int) {
		if state, err := broken.r.Stage("ws1", "a1", number, chunks[number-1]); err != nil || state != Staging {
			t.Fatal(number, state, err)
		}
	}
	for number := 1; number <= 50; number++ {
		send(number)
	}
	if _, gaps, err := broken.r.Seal("ws1", "a1"); err != ErrRejected || !reflect.DeepEqual(gaps[:2], []Gap{{What: GapChunks}, {What: GapRows}}) {
		t.Fatal(gaps, err)
	}
	for number := 31; number <= 80; number++ {
		send(number)
	}
	order := random.Perm(chunkCount)
	for _, index := range order {
		if index >= 80 || index%3 == 0 {
			send(index + 1)
		}
	}
	for _, index := range order[:20] {
		send(index + 1)
	}

	sorted := append([]Record(nil), every...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Key < sorted[j].Key })
	var encoded, content []byte
	for _, record := range sorted {
		encoded = append(encoded, secondEncoding(record.Key, record.Amount, record.Currency, record.Digest)...)
		content = append(content, secondEncoding("event", "", record.Key, record.Amount, record.Currency, record.Digest)...)
	}
	recordSum, contentSum := sha256.Sum256(encoded), sha256.Sum256(content)
	wantDigest, wantContent := hex.EncodeToString(recordSum[:]), hex.EncodeToString(contentSum[:])

	for name, h := range map[string]*harness{"straight": straight, "broken": broken} {
		staged, err := h.r.Staged("ws1", "a1")
		if err != nil || staged.Rows != 120000 || staged.Chunks != chunkCount || !staged.Complete || !reflect.DeepEqual(staged.Sums, declared) || staged.Digest != wantDigest {
			t.Fatalf("%s staged: %+v %v", name, staged, err)
		}
		if state, gaps, err := h.r.Seal("ws1", "a1"); err != nil || state != Sealed {
			t.Fatal(name, gaps, err)
		}
		view := h.view("a1")
		if view.Rows != 120000 || view.Digest != wantDigest || !reflect.DeepEqual(view.Sums, declared) || view.Spec.Bytes != spec.Bytes {
			t.Fatalf("%s sealed: %+v", name, view)
		}
		if out := h.publish(october, 0); out.Number != 1 || !out.Written {
			t.Fatalf("%s: %+v", name, out)
		}
		manifest := h.manifest(october, 1)
		if manifest.Records != 120000 || manifest.Content != wantContent || !reflect.DeepEqual(manifest.Sums, declared) {
			t.Fatalf("%s manifest: %+v", name, manifest)
		}
		records := h.all(october, 1)
		if len(records) != 120000 {
			t.Fatal(len(records))
		}
		for index, record := range records {
			if record.Key != sorted[index].Key || record.Amount != sorted[index].Amount || record.Currency != sorted[index].Currency || record.Digest != sorted[index].Digest || record.Artifact != "a1" {
				t.Fatalf("%s record %d: %+v", name, index, record)
			}
		}
	}

	changed := append([]Record(nil), chunks[119]...)
	changed[999] = rec(changed[999].Key, "0.0001", changed[999].Currency)
	other := spec
	other.ID, other.Bytes = "a2", digestOf("other bytes")
	other.Sums = map[string]string{}
	for currency, sum := range sums {
		if currency == changed[999].Currency {
			sum += 1 - scaled(t, chunks[119][999].Amount)
		}
		other.Sums[currency] = unscaled(sum)
	}
	broken.register(other)
	for number := chunkCount; number >= 1; number-- {
		chunk := chunks[number-1]
		if number == chunkCount {
			chunk = changed
		}
		if _, err := broken.r.Stage("ws1", "a2", number, chunk); err != nil {
			t.Fatal(err)
		}
	}
	if state, gaps, err := broken.r.Seal("ws1", "a2"); err != nil || state != Sealed {
		t.Fatal(gaps, err)
	}
	broken.blocked(october, 1, 1, Blocker{What: InConflict, Document: "d1"})
	if manifest := broken.manifest(october, 1); manifest.Content != wantContent {
		t.Fatal("a published manifest changed")
	}
}

type truthRow struct {
	amount   string
	currency string
	digest   string
}

func TestManifestsHoldExactlyWhatTheProviderFinallyStates(t *testing.T) {
	random := rand.New(rand.NewSource(1861))
	amounts := []string{"1", "2.5", "-3", "0.0001", "100", "0", "7.25", "-0.5"}
	currencies := []string{"USD", "EUR", "KRW"}
	for round := 0; round < 60; round++ {
		h := newHarness(t, func(l *Limits) { l.ChunkRecords = 4; l.Page = 7 })
		subjects := []Subject{october, november, {WorkspaceID: "ws1", Provider: "gcp", Scope: "billing-1", Period: "2026-10"}, {WorkspaceID: "ws2", Provider: "aws", Scope: "payer-1", Period: "2026-10"}}
		truth := map[Subject][]string{}
		type delivery struct {
			spec   Spec
			chunks [][]Record
		}
		var deliveries []delivery
		moved := []truthRow{}
		documents := 3 + random.Intn(6)
		for number := 0; number < documents; number++ {
			at := subjects[random.Intn(len(subjects))]
			document := fmt.Sprintf("doc-%d", number)
			kind := EventKeys
			if random.Intn(2) == 0 {
				kind = LocatorKeys
			}
			versions := 1 + random.Intn(4)
			var current []truthRow
			for count := random.Intn(9); count > 0; count-- {
				current = append(current, truthRow{amounts[random.Intn(len(amounts))], currencies[random.Intn(len(currencies))], digestOf(document, fmt.Sprint(random.Int63()))[:24]})
			}
			var ids []string
			for index := range current {
				ids = append(ids, fmt.Sprintf("%s-e%d", document, index))
			}
			next := len(current)
			for version := 1; version <= versions; version++ {
				if version > 1 {
					switch random.Intn(6) {
					case 0:
						current, ids = nil, nil
					case 1:
						if len(current) > 0 {
							index := random.Intn(len(current))
							moved = append(moved, current[index])
							current = append(current[:index:index], current[index+1:]...)
							ids = append(ids[:index:index], ids[index+1:]...)
						}
					case 2:
						if len(current) > 0 {
							index := random.Intn(len(current))
							current[index].amount = amounts[random.Intn(len(amounts))]
							current[index].digest = digestOf(document, "corrected", fmt.Sprint(random.Int63()))[:24]
						}
					case 3:
						if len(moved) > 0 {
							current = append(current, moved[len(moved)-1])
							moved = moved[:len(moved)-1]
							ids = append(ids, fmt.Sprintf("%s-e%d", document, next))
							next++
						}
					default:
						current = append(current, truthRow{amounts[random.Intn(len(amounts))], currencies[random.Intn(len(currencies))], digestOf(document, fmt.Sprint(random.Int63()))[:24]})
						ids = append(ids, fmt.Sprintf("%s-e%d", document, next))
						next++
					}
				}
				var records []Record
				for index, row := range current {
					key := ids[index]
					if kind == LocatorKeys {
						key = fmt.Sprintf("line-%d", index+1)
					}
					records = append(records, Record{Key: key, Digest: row.digest, Amount: row.amount, Currency: row.currency})
				}
				var chunks [][]Record
				for from := 0; from < len(records); {
					size := 1 + random.Intn(4)
					if size > len(records)-from {
						size = len(records) - from
					}
					chunks = append(chunks, records[from:from+size])
					from += size
				}
				id := fmt.Sprintf("%s-v%d", document, version)
				spec := base(id, document, uint64(version))
				if len(records) > 0 {
					spec = original(t, id, document, uint64(version), chunks...)
				}
				spec.Subject, spec.Kind = at, kind
				deliveries = append(deliveries, delivery{spec, chunks})
			}
			for _, row := range current {
				truth[at] = append(truth[at], row.amount+" "+row.currency+" "+row.digest)
			}
			if _, known := truth[at]; !known {
				truth[at] = nil
			}
		}
		published := map[Subject]uint64{}
		random.Shuffle(len(deliveries), func(i, j int) { deliveries[i], deliveries[j] = deliveries[j], deliveries[i] })
		publish := func(at Subject, final bool) {
			out, err := h.r.Publish(at, published[at])
			switch {
			case err == nil:
				published[at] = out.Number
			case err == ErrNotFound && !final:
			case err == ErrRejected && out.BlockerTotal == 0 && published[at] == 0:
			default:
				t.Fatalf("round %d publish %v: %+v %v", round, at, out, err)
			}
		}
		for _, made := range deliveries {
			h.register(made.spec)
			for _, index := range random.Perm(len(made.chunks)) {
				if _, err := h.r.Stage(made.spec.Subject.WorkspaceID, made.spec.ID, index+1, made.chunks[index]); err != nil {
					t.Fatal(err)
				}
			}
			if state, gaps, err := h.r.Seal(made.spec.Subject.WorkspaceID, made.spec.ID); err != nil || state != Sealed {
				t.Fatal(gaps, err)
			}
			if random.Intn(3) == 0 {
				publish(made.spec.Subject, false)
			}
		}
		for at, want := range truth {
			publish(at, true)
			var got []string
			if published[at] > 0 {
				for _, record := range h.all(at, published[at]) {
					got = append(got, record.Amount+" "+record.Currency+" "+record.Digest)
				}
			}
			normal := make([]string, len(want))
			for index, row := range want {
				var amount, currency, digest string
				fmt.Sscan(row, &amount, &currency, &digest)
				normal[index] = unscaled(scaled(t, amount)) + " " + currency + " " + digest
			}
			sort.Strings(got)
			sort.Strings(normal)
			if !reflect.DeepEqual(got, normal) && (len(got) > 0 || len(normal) > 0) {
				t.Fatalf("round %d subject %v:\n got %v\nwant %v", round, at, got, normal)
			}
			if status, err := h.r.Status(at); err != nil || status.BlockerTotal != 0 || status.Behind.Total != 0 || status.Staging != 0 || status.Differs != Same {
				t.Fatalf("round %d: %+v %v", round, status, err)
			}
		}
	}
}
