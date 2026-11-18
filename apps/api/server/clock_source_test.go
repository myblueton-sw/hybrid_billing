package server

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var wallCalls = regexp.MustCompile(`\b(time\.(Now|Since|Until|After|AfterFunc|Tick|NewTicker|NewTimer)|context\.(WithDeadline|WithTimeout))\(`)

func TestOneServerClock(t *testing.T) {
	files, _ := filepath.Glob("*.go")
	more, _ := filepath.Glob("../cmd/hb-api/*.go")
	allowed := map[string]int{"clock.go": 4, "../cmd/hb-api/main.go": 1}
	for _, name := range append(files, more...) {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		found := wallCalls.FindAllString(string(data), -1)
		if len(found) != allowed[name] {
			t.Errorf("%s calls the wall clock %v", name, found)
		}
	}
	main, _ := os.ReadFile("../cmd/hb-api/main.go")
	if !strings.Contains(string(main), "context.WithTimeout(context.Background(), 5*time.Second)") {
		t.Error("the shutdown deadline moved")
	}
}
