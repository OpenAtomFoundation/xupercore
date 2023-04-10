package utxo

import (
	"math/big"
	"reflect"
	"testing"
)

func TestUtxoItem_Loads(t *testing.T) {
	type fields struct {
		Amount       *big.Int
		FrozenHeight int64
	}
	type args struct {
		data []byte
	}
	var (
		tests = []struct {
			name    string
			fields  fields
			args    args
			wantErr bool
		}{
			{
				name:   "invalid data",
				fields: fields{},
				args: args{
					data: []byte("[]"),
				},
				wantErr: true,
			},
		}
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &UtxoItem{
				Amount:       tt.fields.Amount,
				FrozenHeight: tt.fields.FrozenHeight,
			}
			if err := i.Loads(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("UtxoItem.Loads() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUtxoItem_Dumps_And_Loads(t *testing.T) {
	type fields struct {
		Amount       *big.Int
		FrozenHeight int64
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name:   "empty UTXO",
			fields: fields{},
		},
		{
			name: "normal UTXO",
			fields: fields{
				Amount:       big.NewInt(100),
				FrozenHeight: 100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &UtxoItem{
				Amount:       tt.fields.Amount,
				FrozenHeight: tt.fields.FrozenHeight,
			}
			dumpI, err := i.Dumps()
			if (err != nil) != tt.wantErr {
				t.Errorf("UtxoItem.Dumps() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			loadI := &UtxoItem{}
			err = loadI.Loads(dumpI)
			if err != nil {
				t.Fatalf("UtxoItem.Loads() = %v, want nil", err)
			}
			if !reflect.DeepEqual(loadI, i) {
				t.Errorf("UtxoItem.Loads() = %v, want %v", loadI, i)
			}
		})
	}
}

func TestUtxoItem_IsFrozen(t *testing.T) {
	type fields struct {
		Amount       *big.Int
		FrozenHeight int64
	}
	type args struct {
		curHeight int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "height -1",
			fields: fields{
				FrozenHeight: -1,
			},
			args: args{
				curHeight: 10,
			},
			want: true,
		},
		{
			name: "frozen height over current",
			fields: fields{
				FrozenHeight: 11,
			},
			args: args{
				curHeight: 10,
			},
			want: true,
		},
		{
			name: "frozen height equal current",
			fields: fields{
				FrozenHeight: 10,
			},
			args: args{
				curHeight: 10,
			},
			want: false,
		},
		{
			name: "frozen height less current",
			fields: fields{
				FrozenHeight: 9,
			},
			args: args{
				curHeight: 10,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &UtxoItem{
				Amount:       tt.fields.Amount,
				FrozenHeight: tt.fields.FrozenHeight,
			}
			if got := i.IsFrozen(tt.args.curHeight); got != tt.want {
				t.Errorf("UtxoItem.IsFrozen() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUtxoItem_IsEmpty(t *testing.T) {
	type fields struct {
		Amount       *big.Int
		FrozenHeight int64
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "empty",
			fields: fields{},
			want:   true,
		},
		{
			name: "0",
			fields: fields{
				Amount: big.NewInt(0),
			},
			want: false,
		},
		{
			name: "non-0",
			fields: fields{
				Amount: big.NewInt(100),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &UtxoItem{
				Amount:       tt.fields.Amount,
				FrozenHeight: tt.fields.FrozenHeight,
			}
			if got := i.IsEmpty(); got != tt.want {
				t.Errorf("UtxoItem.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewUtxoItem(t *testing.T) {
	type args struct {
		amount       []byte
		frozenHeight int64
	}
	tests := []struct {
		name string
		args args
		want *UtxoItem
	}{
		{
			name: "0",
			args: args{
				amount:       nil,
				frozenHeight: 0,
			},
			want: &UtxoItem{
				Amount: big.NewInt(0),
			},
		},
		{
			name: "non-0",
			args: args{
				amount:       big.NewInt(100).Bytes(),
				frozenHeight: 10,
			},
			want: &UtxoItem{
				Amount:       big.NewInt(100),
				FrozenHeight: 10,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewUtxoItem(tt.args.amount, tt.args.frozenHeight); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewUtxoItem() = %v, want %v", got, tt.want)
			}
		})
	}
}
