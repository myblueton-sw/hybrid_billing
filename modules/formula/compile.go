package formula

import (
	"sort"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type step struct {
	kind     string
	value    fraction
	variable int
	formula  int
	left     *step
	right    *step
}

type unit struct {
	id     string
	output Type
	scale  int
	mode   string
	root   *step
	refs   []int
}

type Compiled struct {
	profile   decimal.Profile
	limits    Limits
	variables map[string]int
	units     []unit
	order     []int
	targets   map[string]int
}

type findings map[Finding]bool

func (f findings) add(code, formula string) { f[Finding{Code: code, Formula: formula}] = true }

func (f findings) list() []Finding {
	out := make([]Finding, 0, len(f))
	for finding := range f {
		out = append(out, finding)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].Code != out[b].Code {
			return codeOrder[out[a].Code] < codeOrder[out[b].Code]
		}
		return out[a].Formula < out[b].Formula
	})
	return out
}

type walk struct {
	profile   decimal.Profile
	limits    Limits
	count     int
	malformed bool
}

func (w *walk) form(node Node) bool {
	switch {
	case node.Kind == NodeConstant:
		if len(node.Number) > maxNumberBytes || node.Name != "" || node.Formula != "" || !validType(node.Type) {
			return false
		}
		_, err := decimal.Parse(w.profile, node.Number)
		return err == nil
	case node.Kind == NodeVariable:
		return validID(node.Name) && node.Number == "" && node.Formula == "" && node.Type == Type{}
	case node.Kind == NodeReference:
		return validID(node.Formula) && node.Number == "" && node.Name == "" && node.Type == Type{}
	case operations[node.Kind]:
		return len(node.Operands) == 2 && node.Number == "" && node.Name == "" && node.Formula == "" && node.Type == Type{}
	}
	return false
}

func (w *walk) copy(node Node, depth int) (Node, string) {
	w.count++
	if w.count > w.limits.Nodes {
		return Node{}, CodeTooLarge
	}
	if depth > w.limits.Depth {
		return Node{}, CodeTooDeep
	}
	out := Node{Kind: node.Kind, Number: node.Number, Type: node.Type, Name: node.Name, Formula: node.Formula}
	if !w.form(node) || (!operations[node.Kind] && len(node.Operands) > 0) {
		w.malformed = true
	}
	if operations[node.Kind] && len(node.Operands) == 2 {
		out.Operands = make([]Node, 0, 2)
		for _, operand := range node.Operands {
			child, code := w.copy(operand, depth+1)
			if code != "" {
				return Node{}, code
			}
			out.Operands = append(out.Operands, child)
		}
	}
	return out, ""
}

func structure(profile decimal.Profile, limits Limits, set Set) (Set, []Finding) {
	found := findings{}
	if len(set.Variables) > limits.Variables || len(set.Formulas) > limits.Formulas {
		found.add(CodeTooMany, "")
		return Set{}, found.list()
	}
	out := Set{Variables: make([]Variable, 0, len(set.Variables)), Formulas: make([]Formula, 0, len(set.Formulas))}
	names := map[string]bool{}
	for _, variable := range set.Variables {
		if !validID(variable.Name) || !validType(variable.Type) || names[variable.Name] {
			found.add(CodeMalformed, "")
			continue
		}
		names[variable.Name] = true
		out.Variables = append(out.Variables, variable)
	}
	ids := map[string]bool{}
	for _, formula := range set.Formulas {
		if !validID(formula.ID) {
			found.add(CodeMalformed, "")
			continue
		}
		if ids[formula.ID] {
			found.add(CodeMalformed, "")
		}
		ids[formula.ID] = true
		w := &walk{profile: profile, limits: limits}
		root, code := w.copy(formula.Root, 1)
		if code != "" {
			found.add(code, formula.ID)
			continue
		}
		_, known := quanta[formula.Quantum]
		readable := false
		if known {
			_, err := decimal.Parse(profile, formula.Quantum)
			readable = err == nil
		}
		if w.malformed || !validID(formula.Target) || formula.Rank < 1 || formula.Rank > maxRank ||
			!validType(formula.Output) || !readable || !modes[formula.Mode] {
			found.add(CodeMalformed, formula.ID)
		}
		copied := formula
		copied.Root = root
		out.Formulas = append(out.Formulas, copied)
	}
	if len(found) > 0 {
		return Set{}, found.list()
	}
	sort.Slice(out.Variables, func(a, b int) bool { return out.Variables[a].Name < out.Variables[b].Name })
	sort.Slice(out.Formulas, func(a, b int) bool { return out.Formulas[a].ID < out.Formulas[b].ID })
	return out, nil
}

func rule(kind string, left, right Type) (Type, bool) {
	switch kind {
	case NodeAdd, NodeSub, NodeMin, NodeMax:
		return left, left == right
	case NodeMul:
		switch {
		case left.Kind == KindNumber:
			return right, true
		case right.Kind == KindNumber:
			return left, true
		case left.Kind == KindQuantity && right.Kind == KindPrice && left.Unit == right.Unit:
			return Type{Kind: KindMoney, Currency: right.Currency}, true
		case left.Kind == KindPrice && right.Kind == KindQuantity && left.Unit == right.Unit:
			return Type{Kind: KindMoney, Currency: left.Currency}, true
		}
	case NodeDiv:
		switch {
		case right.Kind == KindNumber:
			return left, true
		case left == right:
			return Type{Kind: KindNumber}, true
		case left.Kind == KindMoney && right.Kind == KindQuantity:
			return Type{Kind: KindPrice, Currency: left.Currency, Unit: right.Unit}, true
		case left.Kind == KindMoney && right.Kind == KindPrice && left.Currency == right.Currency:
			return Type{Kind: KindQuantity, Unit: right.Unit}, true
		}
	case NodeGrossUp:
		return left, right.Kind == KindNumber
	}
	return Type{}, false
}

type builder struct {
	profile   decimal.Profile
	variables map[string]Type
	slots     map[string]int
	outputs   map[string]Type
	indexes   map[string]int
	found     findings
	id        string
	refs      map[int]bool
}

func (b *builder) build(node Node) (*step, Type, bool) {
	switch node.Kind {
	case NodeConstant:
		value, _ := decimal.Parse(b.profile, node.Number)
		return &step{kind: NodeConstant, value: of(value)}, node.Type, true
	case NodeVariable:
		declared, ok := b.variables[node.Name]
		if !ok {
			b.found.add(CodeUnknownName, b.id)
			return nil, Type{}, false
		}
		return &step{kind: NodeVariable, variable: b.slots[node.Name]}, declared, true
	case NodeReference:
		output, ok := b.outputs[node.Formula]
		if !ok {
			b.found.add(CodeUnknownName, b.id)
			return nil, Type{}, false
		}
		b.refs[b.indexes[node.Formula]] = true
		return &step{kind: NodeReference, formula: b.indexes[node.Formula]}, output, true
	}
	left, leftType, leftTyped := b.build(node.Operands[0])
	right, rightType, rightTyped := b.build(node.Operands[1])
	if node.Operands[1].Kind == NodeConstant {
		value, _ := decimal.Parse(b.profile, node.Operands[1].Number)
		margin := of(value)
		if node.Kind == NodeDiv && margin.sign() == 0 {
			b.found.add(CodeConstantZero, b.id)
		}
		if node.Kind == NodeGrossUp && (margin.sign() < 0 || !margin.less(whole(1))) {
			b.found.add(CodeMarginRange, b.id)
		}
	}
	if !leftTyped || !rightTyped {
		return nil, Type{}, false
	}
	result, ok := rule(node.Kind, leftType, rightType)
	if !ok {
		b.found.add(CodeTypeMismatch, b.id)
		return nil, Type{}, false
	}
	return &step{kind: node.Kind, left: left, right: right}, result, true
}

func Compile(profile decimal.Profile, limits Limits, set Set) (*Compiled, []Finding, error) {
	if !profile.Valid() || !limits.valid() {
		return nil, nil, ErrInvalid
	}
	copied, structural := structure(profile, limits, set)
	if structural != nil {
		return nil, structural, nil
	}
	return compile(profile, limits, copied)
}

func compile(profile decimal.Profile, limits Limits, set Set) (*Compiled, []Finding, error) {
	b := &builder{profile: profile, variables: map[string]Type{}, slots: map[string]int{}, outputs: map[string]Type{}, indexes: map[string]int{}, found: findings{}}
	for index, variable := range set.Variables {
		b.variables[variable.Name] = variable.Type
		b.slots[variable.Name] = index
	}
	for index, formula := range set.Formulas {
		b.outputs[formula.ID] = formula.Output
		b.indexes[formula.ID] = index
	}
	type place struct {
		target string
		rank   int
	}
	places := map[place]int{}
	compiled := &Compiled{profile: profile, limits: limits, variables: b.slots, units: make([]unit, len(set.Formulas)), targets: map[string]int{}}
	for index, formula := range set.Formulas {
		b.id, b.refs = formula.ID, map[int]bool{}
		root, output, typed := b.build(formula.Root)
		if typed && output != formula.Output {
			b.found.add(CodeTypeMismatch, formula.ID)
		}
		refs := make([]int, 0, len(b.refs))
		for ref := range b.refs {
			refs = append(refs, ref)
		}
		sort.Ints(refs)
		compiled.units[index] = unit{id: formula.ID, output: formula.Output, scale: quanta[formula.Quantum], mode: formula.Mode, root: root, refs: refs}
		places[place{formula.Target, formula.Rank}]++
		if best, ok := compiled.targets[formula.Target]; !ok || formula.Rank < set.Formulas[best].Rank {
			compiled.targets[formula.Target] = index
		}
	}
	waiting := make([]int, len(compiled.units))
	dependents := make([][]int, len(compiled.units))
	ready := []int{}
	for index, u := range compiled.units {
		waiting[index] = len(u.refs)
		for _, ref := range u.refs {
			dependents[ref] = append(dependents[ref], index)
		}
		if len(u.refs) == 0 {
			ready = append(ready, index)
		}
	}
	for len(ready) > 0 {
		index := ready[0]
		ready = ready[1:]
		compiled.order = append(compiled.order, index)
		for _, dependent := range dependents[index] {
			waiting[dependent]--
			if waiting[dependent] == 0 {
				ready = append(ready, dependent)
			}
		}
	}
	for index, left := range waiting {
		if left > 0 {
			b.found.add(CodeCycle, compiled.units[index].id)
		}
	}
	for _, formula := range set.Formulas {
		if places[place{formula.Target, formula.Rank}] > 1 {
			b.found.add(CodeRankConflict, formula.ID)
		}
	}
	if len(b.found) > 0 {
		return nil, b.found.list(), nil
	}
	return compiled, nil, nil
}
