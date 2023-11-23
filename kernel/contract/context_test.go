package contract

import "testing"

func TestResponse_HasError(t *testing.T) {
	type fields struct {
		Status  int
		Message string
		Body    []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "no error",
			fields: fields{
				Status: StatusOK,
			},
			want: false,
		},
		{
			name: "threshold error",
			fields: fields{
				Status: StatusErrorThreshold,
			},
			want: true,
		},
		{
			name: "normal error",
			fields: fields{
				Status: StatusError,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Response{
				Status:  tt.fields.Status,
				Message: tt.fields.Message,
				Body:    tt.fields.Body,
			}
			if got := r.HasError(); got != tt.want {
				t.Errorf("Response.HasError() = %v, want %v", got, tt.want)
			}
		})
	}
}
