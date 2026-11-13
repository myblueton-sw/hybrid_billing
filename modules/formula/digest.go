package formula

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"strconv"
)

type printer struct{ sum hash.Hash }

func (p printer) field(text string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(text)))
	p.sum.Write(size[:])
	p.sum.Write([]byte(text))
}

func (p printer) number(value int) { p.field(strconv.Itoa(value)) }

func (p printer) kind(t Type) {
	p.field(t.Kind)
	p.field(t.Currency)
	p.field(t.Unit)
}

func (p printer) node(node Node) {
	p.field(node.Kind)
	switch node.Kind {
	case NodeConstant:
		p.field(node.Number)
		p.kind(node.Type)
	case NodeVariable:
		p.field(node.Name)
	case NodeReference:
		p.field(node.Formula)
	default:
		p.node(node.Operands[0])
		p.node(node.Operands[1])
	}
}

func (p printer) hex() string { return hex.EncodeToString(p.sum.Sum(nil)) }

func contentDigest(set Set) string {
	p := printer{sum: sha256.New()}
	p.field(Semantics)
	p.number(len(set.Variables))
	for _, variable := range set.Variables {
		p.field(variable.Name)
		p.kind(variable.Type)
	}
	p.number(len(set.Formulas))
	for _, formula := range set.Formulas {
		p.field(formula.ID)
		p.field(formula.Target)
		p.number(formula.Rank)
		p.kind(formula.Output)
		p.field(formula.Quantum)
		p.field(formula.Mode)
		p.node(formula.Root)
	}
	return p.hex()
}

func subjectDigest(workspace, set string, revision int, content string, impact Impact) string {
	p := printer{sum: sha256.New()}
	p.field(subjectLabel)
	p.field(workspace)
	p.field(set)
	p.number(revision)
	p.field(content)
	p.field(impact.Manifest)
	p.field(impact.ManifestDigest)
	p.number(impact.Epoch)
	p.field(impact.Coverage)
	p.number(impact.Evaluated)
	p.number(impact.Changed)
	p.number(impact.WithoutValue)
	p.number(impact.Failed)
	return p.hex()
}

func cloneNode(node Node) Node {
	out := node
	if len(node.Operands) > 0 {
		out.Operands = []Node{cloneNode(node.Operands[0]), cloneNode(node.Operands[1])}
	}
	return out
}

func cloneSet(set Set) Set {
	out := Set{Variables: append([]Variable{}, set.Variables...), Formulas: make([]Formula, len(set.Formulas))}
	for index, formula := range set.Formulas {
		out.Formulas[index] = formula
		out.Formulas[index].Root = cloneNode(formula.Root)
	}
	return out
}
