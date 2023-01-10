package parachain

import "testing"

func TestGroup_hasAccessAuth(t *testing.T) {
	testGroup := &Group{
		Admin:      []string{"address_both", "address_admin"},
		Identities: []string{"address_both", "address_identities"},
	}
	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{
			name:    "both set",
			address: "address_both",
			want:    true,
		},
		{
			name:    "admin",
			address: "address_admin",
			want:    true,
		},
		{
			name:    "identities",
			address: "address_identities",
			want:    true,
		},
		{
			name:    "not set",
			address: "address_not_set",
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testGroup.hasAccessAuth(tt.address); got != tt.want {
				t.Errorf("Group.hasAccessAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroup_hasAdminAuth(t *testing.T) {
	testGroup := &Group{
		Admin: []string{"address_admin"},
	}
	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{
			name:    "admin",
			address: "address_admin",
			want:    true,
		},
		{
			name:    "not set",
			address: "address_not_set",
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := testGroup.hasAdminAuth(tt.address); got != tt.want {
				t.Errorf("Group.hasAdminAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroup_IsParaChainEnable(t *testing.T) {
	type fields struct {
		Status int
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "enabled",
			fields: fields{Status: ParaChainStatusStart},
			want:   true,
		},
		{
			name:   "disabled",
			fields: fields{Status: ParaChainStatusStop},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Group{
				Status: tt.fields.Status,
			}
			if got := g.IsParaChainEnable(); got != tt.want {
				t.Errorf("Group.IsParaChainEnable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_contains(t *testing.T) {
	type args struct {
		items []string
		item  string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "item in list",
			args: args{
				items: []string{"a", "b"},
				item:  "a",
			},
			want: true,
		},
		{
			name: "item not in list",
			args: args{
				items: []string{"a", "b"},
				item:  "c",
			},
			want: false,
		},
		{
			name: "empty list",
			args: args{
				items: []string{},
				item:  "c",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.args.items, tt.args.item); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
