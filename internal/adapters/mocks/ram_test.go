package mocks_test

import (
	"context"
	"testing"
	"time"

	"github.com/RobMil91/free-orgx/internal/adapters/mocks"
	"github.com/RobMil91/free-orgx/internal/ports"
)

func TestRAM_IsValid(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		cookie  string
		users   map[string]ports.User
		want    *ports.User
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name:   "denie_old",
			cookie: "aragorn",
			users: map[string]ports.User{
				"gondor": {
					Cookie: ports.SessionCookie{
						Value:      "aragorn",
						CreateTime: time.Now().Add(-5 * time.Minute),
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: construct the receiver type.
			var r mocks.RAM
			r.Users = tt.users
			got, gotErr := r.IsValid(context.Background(), tt.cookie)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("IsValid() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("IsValid() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
