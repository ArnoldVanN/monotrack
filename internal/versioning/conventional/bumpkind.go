package conventional

type Kind string

const (
	KindMajor Kind = "major"
	KindMinor Kind = "minor"
	KindPatch Kind = "patch"
	KindNone  Kind = ""
)

// DeriveBumpKind picks the highest-precedence bump kind implied by the commits.
// Returns KindNone only if commits is empty.
func DeriveBumpKind(commits []ParsedCommit) Kind {
	if len(commits) == 0 {
		return KindNone
	}

	hasFeat := false
	for _, c := range commits {
		if c.Breaking {
			return KindMajor
		}
		if c.Type == TypeFeat {
			hasFeat = true
		}
	}

	if hasFeat {
		return KindMinor
	}
	return KindPatch
}
