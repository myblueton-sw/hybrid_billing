package support

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

var corpus = []string{
	"MARK-secret-hunter2-0001", "Authorization: Bearer MARK-eyJhbGciOiJIUzI1NiJ9.e30.sig0002", "password=MARK-pass-0003", "MARK-jane.doe0004@acme.example",
	"10.77.0.4-MARK-0005", "fe80::1%eth0-MARK-0006", "MARK-db01.acme.internal-0007", "AKIAMARK0008ABCDEFGH", "ghp_MARK0009abcdefghijklmnopqrstuvwxyz",
	"-----BEGIN PRIVATE KEY-----MARK-0010", "postgres://u:MARK-0011@db/x", "MARK​-zero-width-0012", "MARK-‮-bidi-0013", "MARK-cust-0014-ACME", "MARK-Ärger-0015", "MARK-ärger-0015",
}

func TestBundleProperty(t *testing.T) {
	runs := 300
	if testing.Short() {
		runs = 40
	}
	for seed := 0; seed < runs; seed++ {
		runBundleProperty(t, int64(seed))
	}
}

func runBundleProperty(t *testing.T, seed int64) {
	rng := rand.New(rand.NewSource(seed))
	f := standard(t)
	pseudo := map[string]string{"client": ClassAddress, "contact": ClassEmail, "node": ClassHost, "operator": ClassUser, "subject": ClassIdentifier}
	model := map[string]map[string]int{}
	expected := [][]string{}
	items := []Item{}
	for index := 0; index < 1+rng.Intn(6); index++ {
		item := Item{Source: SourceLog}
		used := map[string]bool{}
		for at := 0; at < 1+rng.Intn(7); at++ {
			marker := corpus[rng.Intn(len(corpus))]
			if rng.Intn(3) == 0 {
				marker = " " + strings.ToUpper(marker[:4]) + marker[4:] + " "
			}
			switch rng.Intn(5) {
			case 0:
				key := []string{"client", "contact", "node", "operator", "subject"}[rng.Intn(5)]
				if !used[key] {
					item.Fields = append(item.Fields, Field{Key: key, Text: marker})
				}
				used[key] = true
			case 1:
				if !used["event"] {
					item.Fields = append(item.Fields, Field{Key: "event", Text: []string{"login.failed", "import.done", marker}[rng.Intn(3)]})
				}
				used["event"] = true
			case 2:
				item.Fields = append(item.Fields, Field{Key: "x" + strconv.Itoa(at), Text: marker})
			case 3:
				item.Fields = append(item.Fields, Field{Key: "login.failed", Text: "login.failed"})
			default:
				if !used["retries"] {
					item.Fields = append(item.Fields, Field{Key: "retries", Int: rng.Int63n(1000)})
				}
				used["retries"] = true
			}
		}
		items = append(items, item)
		keys := map[string]bool{}
		duplicate := false
		for _, field := range item.Fields {
			duplicate = duplicate || keys[field.Key]
			keys[field.Key] = true
		}
		if duplicate {
			continue
		}
		want := []string{}
		for _, field := range item.Fields {
			switch {
			case pseudo[field.Key] != "":
				class := pseudo[field.Key]
				if model[class] == nil {
					model[class] = map[string]int{}
				}
				norm := asciiLower(strings.TrimSpace(field.Text))
				if model[class][norm] == 0 {
					model[class][norm] = len(model[class]) + 1
				}
				want = append(want, field.Key+"="+class+":"+class+"-"+strconv.Itoa(model[class][norm]))
			case field.Key == "event" && (field.Text == "login.failed" || field.Text == "import.done"):
				want = append(want, "event=enum:"+field.Text)
			case field.Key == "retries":
				want = append(want, "retries=integer:"+strconv.FormatInt(field.Int, 10))
			}
		}
		if len(want) > 0 {
			sortStrings(want)
			expected = append(expected, want)
		}
	}
	b, err := f.r.Create("s1", items, ann)
	if len(expected) == 0 {
		reason(t, "empty", err, ReasonEmpty)
		return
	}
	f.must(err)
	for _, marker := range corpus {
		for _, part := range []string{marker, "MARK", strings.TrimSpace(marker)} {
			if strings.Contains(string(b.Bytes), part) {
				t.Fatalf("seed %d: %q leaked", seed, part)
			}
		}
	}
	if len(b.Items) != len(expected) || b.Summary.Kept != len(expected) {
		t.Fatalf("seed %d: %d items, want %d", seed, len(b.Items), len(expected))
	}
	for index, item := range b.Items {
		got := []string{}
		for _, field := range item.Fields {
			got = append(got, field.Key+"="+field.Class+":"+field.Value)
		}
		if strings.Join(got, ",") != strings.Join(expected[index], ",") {
			t.Fatalf("seed %d item %d: %v want %v", seed, index, got, expected[index])
		}
	}
	f.must(second(f.r.Preview("s1", b.ID, ann)))
	data, err := f.r.Export("s1", b.ID, b.Digest, ann)
	f.must(err)
	if sum(data) != b.Digest {
		t.Fatalf("seed %d: digest", seed)
	}
}

func asciiLower(text string) string {
	out := []byte(text)
	for index, c := range out {
		if 'A' <= c && c <= 'Z' {
			out[index] = c + 32
		}
	}
	return string(out)
}

func sortStrings(list []string) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && list[j] < list[j-1]; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}
