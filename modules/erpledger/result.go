package erpledger

type ResultView struct {
	Submission     SubmissionView
	Reconciliation Reconciliation
	Documents      []DocumentView
}

func (r *Registry) ResultOf(workspace, key string) (ResultView, error) {
	r.enter()
	defer r.leave()
	s, sub, err := r.find(workspace, key)
	if err != nil {
		return ResultView{}, err
	}
	return resultView(s, sub), nil
}

func resultView(s *space, sub *submission) ResultView {
	out := ResultView{Submission: sub.copy(), Reconciliation: reconciliation(s, sub)}
	for _, total := range out.Reconciliation.Documents {
		out.Documents = append(out.Documents, s.documents[externalKey(total.Connection, total.External)].copy())
	}
	return out
}
