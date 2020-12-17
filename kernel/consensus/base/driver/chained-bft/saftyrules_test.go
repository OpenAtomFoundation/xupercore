package chained_bft

import "testing"

func TestCalVotesThreshold(t *testing.T) {
	s := DefaultSaftyRules{}
	sum := 3
	for i := 0; i < 3; i++ {
		if s.CalVotesThreshold(i, sum) {
			t.Error("TestCalVotesThreshold error 1")
		}
	}
	sum = 4
	if !s.CalVotesThreshold(3, sum) {
		t.Error("TestCalVotesThreshold error 2")
	}
}
