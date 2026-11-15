package issuance

import (
	"bytes"
	"strconv"
)

var refusedNames = map[string]bool{
	"A": true, "AA": true, "OpenAction": true, "Annots": true, "URI": true, "Launch": true, "JavaScript": true, "JS": true, "Names": true,
	"AcroForm": true, "XFA": true, "SubmitForm": true, "ResetForm": true, "ImportData": true, "GoToR": true, "GoToE": true, "GoTo3DView": true,
	"Named": true, "Hide": true, "SetOCGState": true, "Thread": true, "Threads": true, "Rendition": true, "Movie": true, "Sound": true, "3D": true,
	"RichMedia": true, "RichMediaExecute": true, "EmbeddedFile": true, "EmbeddedFiles": true, "FS": true, "URL": true, "Encrypt": true,
	"ObjStm": true, "XRef": true, "XRefStm": true, "Prev": true,
	"F": true, "FFilter": true, "FDecodeParms": true, "AF": true, "Filespec": true, "UF": true, "EF": true, "Ref": true, "OPI": true, "Alternates": true,
}

var allowedFilters = map[string]bool{"FlateDecode": true, "DCTDecode": true}

type lexeme struct {
	text  string
	depth int
	name  bool
}

type object struct {
	generation int
	offset     int
	lexemes    []lexeme
	streamed   bool
}

type scanner struct {
	data      []byte
	at        int
	depth     int
	open      *object
	section   int
	pending   []word
	trailer   []lexeme
	objects   map[int]*object
	xref      int
	entries   []entry
	startxref int
}

type word struct {
	text   string
	offset int
}

type entry struct {
	offset     int
	generation int
	used       bool
}

const (
	sectionBody = iota
	sectionTrailer
	sectionStart
	sectionDone
)

func white(c byte) bool {
	return c == 0 || c == '\t' || c == '\n' || c == '\f' || c == '\r' || c == ' '
}

func delimiter(c byte) bool {
	return c == '(' || c == ')' || c == '<' || c == '>' || c == '[' || c == ']' || c == '{' || c == '}' || c == '/' || c == '%'
}

func integer(text string) (int, bool) {
	if text == "" || len(text) > 10 {
		return 0, false
	}
	for index := 0; index < len(text); index++ {
		if text[index] < '0' || text[index] > '9' {
			return 0, false
		}
	}
	value, _ := strconv.Atoi(text)
	return value, true
}

func number(l lexeme) (int, bool) {
	if l.name {
		return 0, false
	}
	return integer(l.text)
}

func inspect(data []byte) string {
	if len(data) == 0 || len(data) > maxFileBytes || !bytes.HasPrefix(data, []byte("%PDF-1.")) && !bytes.HasPrefix(data, []byte("%PDF-2.")) {
		return ReasonMalformedOutput
	}
	for _, end := range []string{"\r\n", "\n", "\r"} {
		if bytes.HasSuffix(data, []byte(end)) {
			data = data[:len(data)-len(end)]
			break
		}
	}
	s := &scanner{data: data, objects: map[int]*object{}, xref: -1, startxref: -1}
	if reason := s.run(); reason != "" {
		return reason
	}
	return s.verify()
}

func (s *scanner) run() string {
	for s.at < len(s.data) {
		c := s.data[s.at]
		var reason string
		switch {
		case white(c):
			s.at++
		case c == '%':
			reason = s.comment()
		case c == '/':
			reason = s.name()
		case c == '(':
			reason = s.literal()
		case c == '<':
			reason = s.angle()
		case c == '>':
			if s.at+1 >= len(s.data) || s.data[s.at+1] != '>' || s.depth == 0 {
				return ReasonMalformedOutput
			}
			s.depth--
			s.at += 2
			reason = s.emit(">>", false)
		case c == '[' || c == ']' || c == '{' || c == '}':
			s.at++
			reason = s.emit(string(c), false)
		case c == ')':
			return ReasonMalformedOutput
		default:
			reason = s.word()
		}
		if reason != "" {
			return reason
		}
	}
	if s.section != sectionDone {
		return ReasonMalformedOutput
	}
	return ""
}

func (s *scanner) comment() string {
	start := s.at
	for s.at < len(s.data) && s.data[s.at] != '\n' && s.data[s.at] != '\r' {
		s.at++
	}
	if bytes.Contains(s.data[start:s.at], []byte("EOF")) || bytes.Contains(s.data[start:s.at], []byte("startxref")) {
		return ReasonMalformedOutput
	}
	return ""
}

func (s *scanner) name() string {
	start := s.at
	s.at++
	for s.at < len(s.data) && !white(s.data[s.at]) && !delimiter(s.data[s.at]) {
		s.at++
	}
	text := string(s.data[start+1 : s.at])
	if bytes.IndexByte([]byte(text), '#') >= 0 || refusedNames[text] {
		return ReasonUnsafeOutput
	}
	return s.emit(text, true)
}

func (s *scanner) literal() string {
	nesting := 0
	for s.at < len(s.data) {
		switch s.data[s.at] {
		case '\\':
			s.at++
		case '(':
			nesting++
		case ')':
			nesting--
			if nesting == 0 {
				s.at++
				return s.emit("()", false)
			}
		}
		s.at++
	}
	return ReasonMalformedOutput
}

func (s *scanner) angle() string {
	if s.at+1 < len(s.data) && s.data[s.at+1] == '<' {
		s.depth++
		s.at += 2
		return s.emit("<<", false)
	}
	end := bytes.IndexByte(s.data[s.at:], '>')
	if end < 0 {
		return ReasonMalformedOutput
	}
	s.at += end + 1
	return s.emit("<>", false)
}

func (s *scanner) word() string {
	start := s.at
	for s.at < len(s.data) && !white(s.data[s.at]) && !delimiter(s.data[s.at]) {
		s.at++
	}
	text := string(s.data[start:s.at])
	switch text {
	case "obj":
		return s.begin()
	case "endobj":
		if s.open == nil || s.depth != 0 {
			return ReasonMalformedOutput
		}
		s.open = nil
		return ""
	case "stream":
		return s.stream()
	case "xref":
		return s.table(start)
	case "trailer":
		if s.section != sectionTrailer {
			return ReasonMalformedOutput
		}
		s.section = sectionStart
		return ""
	case "startxref":
		return s.start()
	}
	if s.open == nil && s.section == sectionBody {
		if _, ok := integer(text); !ok || len(s.pending) == 2 {
			return ReasonMalformedOutput
		}
		s.pending = append(s.pending, word{text: text, offset: start})
		return ""
	}
	return s.emit(text, false)
}

func (s *scanner) emit(text string, name bool) string {
	l := lexeme{text: text, depth: s.depth, name: name}
	switch {
	case s.open != nil:
		s.open.lexemes = append(s.open.lexemes, l)
	case s.section == sectionStart:
		s.trailer = append(s.trailer, l)
	default:
		return ReasonMalformedOutput
	}
	return ""
}

func (s *scanner) begin() string {
	if s.open != nil || s.section != sectionBody || len(s.pending) != 2 {
		return ReasonMalformedOutput
	}
	value, _ := integer(s.pending[0].text)
	generation, _ := integer(s.pending[1].text)
	if _, seen := s.objects[value]; seen {
		return ReasonMalformedOutput
	}
	s.open = &object{generation: generation, offset: s.pending[0].offset}
	s.objects[value] = s.open
	s.pending = nil
	return ""
}

func (s *scanner) length() (int, bool) {
	found, value := 0, 0
	for index, l := range s.open.lexemes {
		if !l.name || l.text != "Length" || l.depth != 1 {
			continue
		}
		size, ok := number(s.open.lexemes[index+1])
		if !ok || index+3 < len(s.open.lexemes) && s.open.lexemes[index+3].text == "R" && !s.open.lexemes[index+3].name {
			return 0, false
		}
		found, value = found+1, size
	}
	return value, found == 1
}

func (s *scanner) stream() string {
	if s.open == nil || s.depth != 0 || len(s.open.lexemes) == 0 || s.open.lexemes[len(s.open.lexemes)-1].text != ">>" || s.open.streamed {
		return ReasonMalformedOutput
	}
	size, ok := s.length()
	if !ok {
		return ReasonMalformedOutput
	}
	switch {
	case bytes.HasPrefix(s.data[s.at:], []byte("\r\n")):
		s.at += 2
	case bytes.HasPrefix(s.data[s.at:], []byte("\n")):
		s.at++
	default:
		return ReasonMalformedOutput
	}
	if size > len(s.data)-s.at {
		return ReasonMalformedOutput
	}
	body := s.data[s.at : s.at+size]
	if bytes.Contains(body, []byte("endstream")) {
		return ReasonMalformedOutput
	}
	s.at += size
	for _, end := range []string{"\r\n", "\n", "\r"} {
		if bytes.HasPrefix(s.data[s.at:], []byte(end)) {
			s.at += len(end)
			break
		}
	}
	if !bytes.HasPrefix(s.data[s.at:], []byte("endstream")) {
		return ReasonMalformedOutput
	}
	s.at += len("endstream")
	if s.at < len(s.data) && !white(s.data[s.at]) && !delimiter(s.data[s.at]) {
		return ReasonMalformedOutput
	}
	s.open.lexemes = append(s.open.lexemes, lexeme{text: "endstream"})
	s.open.streamed = true
	return ""
}

func (s *scanner) line() (string, bool) {
	start := s.at
	for s.at < len(s.data) && s.data[s.at] != '\n' && s.data[s.at] != '\r' {
		s.at++
	}
	text := string(s.data[start:s.at])
	switch {
	case bytes.HasPrefix(s.data[s.at:], []byte("\r\n")):
		s.at += 2
	case s.at < len(s.data):
		s.at++
	default:
		return "", false
	}
	return text, true
}

func (s *scanner) table(start int) string {
	if s.open != nil || s.section != sectionBody || len(s.pending) != 0 {
		return ReasonMalformedOutput
	}
	s.xref, s.section = start, sectionTrailer
	if rest, ok := s.line(); !ok || rest != "" {
		return ReasonMalformedOutput
	}
	header, ok := s.line()
	if !ok || len(header) < 3 || header[:2] != "0 " {
		return ReasonMalformedOutput
	}
	count, valid := integer(header[2:])
	if !valid || count < 1 || count > len(s.data)/20 {
		return ReasonMalformedOutput
	}
	for index := 0; index < count; index++ {
		if len(s.data)-s.at < 20 {
			return ReasonMalformedOutput
		}
		row := string(s.data[s.at : s.at+20])
		s.at += 20
		offset, first := integer(row[0:10])
		generation, second := integer(row[11:16])
		if !first || !second || row[10] != ' ' || row[16] != ' ' || row[17] != 'n' && row[17] != 'f' || row[18:] != " \n" && row[18:] != "\r\n" && row[18:] != " \r" {
			return ReasonMalformedOutput
		}
		s.entries = append(s.entries, entry{offset: offset, generation: generation, used: row[17] == 'n'})
	}
	return ""
}

func (s *scanner) start() string {
	if s.section != sectionStart || s.depth != 0 {
		return ReasonMalformedOutput
	}
	for _, end := range []string{"\r\n", "\n", "\r"} {
		if bytes.HasPrefix(s.data[s.at:], []byte(end)) {
			s.at += len(end)
			break
		}
	}
	begin := s.at
	for s.at < len(s.data) && s.data[s.at] >= '0' && s.data[s.at] <= '9' {
		s.at++
	}
	value, ok := integer(string(s.data[begin:s.at]))
	if !ok {
		return ReasonMalformedOutput
	}
	rest := s.data[s.at:]
	for _, end := range []string{"\r\n", "\n", "\r"} {
		if bytes.HasPrefix(rest, []byte(end)) {
			rest = rest[len(end):]
			break
		}
	}
	if string(rest) != "%%EOF" {
		return ReasonMalformedOutput
	}
	s.startxref, s.section, s.at = value, sectionDone, len(s.data)
	return ""
}

func (s *scanner) verify() string {
	if s.startxref != s.xref || s.entries[0].used || len(s.entries) != len(s.objects)+free(s.entries) {
		return ReasonMalformedOutput
	}
	for index, e := range s.entries {
		o, seen := s.objects[index]
		if e.used != seen || seen && (o.offset != e.offset || o.generation != e.generation) {
			return ReasonMalformedOutput
		}
	}
	for _, o := range s.objects {
		if reason := s.references(o.lexemes); reason != "" {
			return reason
		}
	}
	if reason := s.references(s.trailer); reason != "" {
		return reason
	}
	size, sized := s.value(s.trailer, "Size")
	root, rooted := s.target(s.trailer, "Root")
	if !sized || size != len(s.entries) || !rooted {
		return ReasonMalformedOutput
	}
	catalog := s.objects[root].lexemes
	pages, paged := s.target(catalog, "Pages")
	if !paged || !s.typed(catalog, "Catalog") || !s.typed(s.objects[pages].lexemes, "Pages") {
		return ReasonMalformedOutput
	}
	return ""
}

func (s *scanner) references(list []lexeme) string {
	for index, l := range list {
		if l.name && l.text == "Filter" {
			if !filters(list[index+1:]) {
				return ReasonUnsafeOutput
			}
		}
		if l.name || l.text != "R" {
			continue
		}
		if index < 2 {
			return ReasonMalformedOutput
		}
		value, first := number(list[index-2])
		generation, second := number(list[index-1])
		o, seen := s.objects[value]
		if !first || !second || !seen || o.generation != generation {
			return ReasonMalformedOutput
		}
	}
	return ""
}

func filters(rest []lexeme) bool {
	if len(rest) == 0 {
		return false
	}
	if rest[0].name {
		return allowedFilters[rest[0].text]
	}
	if rest[0].text != "[" {
		return false
	}
	for _, l := range rest[1:] {
		if l.text == "]" && !l.name {
			return true
		}
		if !l.name || !allowedFilters[l.text] {
			return false
		}
	}
	return false
}

func key(list []lexeme, name string) (int, bool) {
	if len(list) == 0 || list[0].text != "<<" {
		return 0, false
	}
	found, at := 0, 0
	for index, l := range list {
		if l.text == ">>" && l.depth == 0 {
			break
		}
		if l.name && l.text == name && l.depth == 1 {
			found, at = found+1, index
		}
	}
	return at, found == 1 && at+1 < len(list)
}

func (s *scanner) value(list []lexeme, name string) (int, bool) {
	at, ok := key(list, name)
	if !ok {
		return 0, false
	}
	return number(list[at+1])
}

func (s *scanner) target(list []lexeme, name string) (int, bool) {
	at, ok := key(list, name)
	if !ok || at+3 >= len(list) || list[at+3].text != "R" || list[at+3].name {
		return 0, false
	}
	return number(list[at+1])
}

func (s *scanner) typed(list []lexeme, kind string) bool {
	at, ok := key(list, "Type")
	return ok && list[at+1].name && list[at+1].text == kind
}

func free(entries []entry) int {
	count := 0
	for _, e := range entries {
		if !e.used {
			count++
		}
	}
	return count
}
