// Package classifier disambiguates a provider's "turn ended" signal into
// either "finished" or "asked you something". The heuristic sits behind the
// Classifier interface so it can be swapped without touching adapters or
// the state engine.
package classifier

import "strings"

type Verdict string

const (
	VerdictDone    Verdict = "done"
	VerdictWaiting Verdict = "waiting"
)

type Classifier interface {
	Classify(lastMessage string) Verdict
}

// Rules is a heuristic implementation over the last assistant message: no
// API calls, no keys required.
type Rules struct{}

func NewRules() Rules { return Rules{} }

var waitingPhrases = []string{
	"let me know", "which one", "should i", "shall i", "do you want",
	"could you confirm", "please confirm", "waiting for your", "your call",
	"how would you like", "which approach", "any preference", "what would you like",
	"can you clarify", "want me to",
}

func (Rules) Classify(lastMessage string) Verdict {
	msg := strings.TrimSpace(lastMessage)
	if msg == "" {
		return VerdictDone
	}
	if strings.HasSuffix(msg, "?") {
		return VerdictWaiting
	}
	lower := strings.ToLower(msg)
	for _, phrase := range waitingPhrases {
		if strings.Contains(lower, phrase) {
			return VerdictWaiting
		}
	}
	return VerdictDone
}
