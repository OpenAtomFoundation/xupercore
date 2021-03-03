package chained_bft

import "testing"

func TestCalVotesThreshold(t *testing.T) {
	s := DefaultSaftyRules{}
	sum := 3
	if s.CalVotesThreshold(1, sum) {
		t.Error("TestCalVotesThreshold error 1")
	}
	sum = 4
	if !s.CalVotesThreshold(3, sum) {
		t.Error("TestCalVotesThreshold error 2")
	}
}
