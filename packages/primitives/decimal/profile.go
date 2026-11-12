package decimal

type ProfileSpec struct {
	ID                  string
	Revision            string
	InputIntegerDigits  int
	InputScale          int
	ResultIntegerDigits int
	ResultScale         int
	IntermediateDigits  int
}

type Profile struct {
	spec  ProfileSpec
	valid bool
}

func NewProfile(spec ProfileSpec) (Profile, error) {
	if spec.ID == "" || spec.Revision == "" ||
		spec.InputIntegerDigits <= 0 || spec.InputScale <= 0 ||
		spec.ResultIntegerDigits < spec.InputIntegerDigits || spec.ResultScale < spec.InputScale ||
		spec.IntermediateDigits < spec.ResultIntegerDigits+spec.ResultScale {
		return Profile{}, ErrInvalidProfile
	}
	return Profile{spec: spec, valid: true}, nil
}

func (p Profile) ID() string { return p.spec.ID }

func (p Profile) Revision() string { return p.spec.Revision }

func (p Profile) IntermediateDigits() int { return p.spec.IntermediateDigits }

func (p Profile) Valid() bool { return p.valid }
