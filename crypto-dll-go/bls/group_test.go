package bls

import (
	"math/big"
	"testing"
)

func TestGroup_calcThreshold(t *testing.T) {
	cases := []struct {
		desc    string
		Ratio   *big.Rat
		want    int
		wantErr bool
	}{
		{
			desc:    "ratio is 0",
			Ratio:   new(big.Rat),
			wantErr: true,
		},
		{
			desc:  "ratio is 1/3",
			Ratio: big.NewRat(1, 3),
			want:  1,
		},
		{
			desc:  "ratio is 2/3",
			Ratio: big.NewRat(2, 3),
			want:  2,
		},
		{
			desc:  "ratio is 1/2",
			Ratio: big.NewRat(1, 2),
			want:  2,
		},
		{
			desc:  "ratio is 1",
			Ratio: big.NewRat(1, 1),
			want:  3,
		},
		{
			desc:    "ratio > 1",
			Ratio:   big.NewRat(2, 1),
			wantErr: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.desc, func(t *testing.T) {
			g := &Group{
				Members: map[string]*Account{
					"1": nil,
					"2": nil,
					"3": nil,
				},
			}
			err := g.calcThreshold(tt.Ratio)
			if (err != nil) != tt.wantErr {
				t.Fatalf("calcThreshold() error = %v, wantErr %v", err, tt.wantErr)
			}
			if g, w := g.Threshold, tt.want; g != w {
				t.Errorf("%s: got %d, want %d", tt.desc, g, w)
			}
		})
	}
}
