package server

import (
	"encoding/json"
	"strings"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

type financialERPResultContext struct {
	Kind                string `json:"kind"`
	Position            string `json:"position"`
	Predecessor         string `json:"predecessor"`
	PredecessorDigest   string `json:"predecessor_digest"`
	PredecessorRevision int    `json:"predecessor_revision"`
	PredecessorReceipt  string `json:"predecessor_receipt"`
	CurrentResult       string `json:"current_result"`
	CurrentState        string `json:"current_state"`
}

func (a *financialERP) resultApplicationContext(b financialERPResultBasis) (string, error) {
	if err := a.validateResultChains(); err != nil {
		return "", err
	}
	external := a.resultExternals[[3]string{b.Workspace, b.Packet.Connection, b.Packet.External}]
	preparation := a.resultPreparations[[2]string{b.Workspace, b.Packet.Preparation}]
	if external != preparation {
		return "", commands.Fail(commands.EvidenceStale)
	}
	c := financialERPResultContext{Kind: "initial", Position: "not_started", CurrentState: "none", CurrentResult: external}
	app, owned := a.resultApplications[[2]string{b.Workspace, b.ID}]
	predecessor := external
	if owned {
		predecessor = app.predecessor
		c.Position = "historical"
		if external == b.ID {
			c.Position = "current"
		}
	}
	if external != "" {
		current := a.resultApplications[[2]string{b.Workspace, external}]
		c.CurrentState = "unknown"
		if current.receipt != "" {
			c.CurrentState = "recorded"
		}
	}
	if predecessor != "" {
		parent := a.resultApplications[[2]string{b.Workspace, predecessor}]
		c.Kind, c.Predecessor, c.PredecessorDigest, c.PredecessorRevision, c.PredecessorReceipt = "replacement", predecessor, parent.digest, parent.document.Revision, parent.receipt
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return "", commands.Fail(commands.PrerequisiteFailed)
	}
	return string(raw), nil
}

func maximumResultApplicationContext() string {
	id, hash := strings.Repeat("r", 128), strings.Repeat("d", 64)
	raw, _ := json.Marshal(financialERPResultContext{Kind: "replacement", Position: "not_started", Predecessor: id, PredecessorDigest: hash, PredecessorRevision: 2147483647, PredecessorReceipt: hash, CurrentResult: id, CurrentState: "recorded"})
	return string(raw)
}
