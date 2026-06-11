package utils

import (
	"strings"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	t.Setenv("NBU1_HOSTNAME", "nbu-master.example.com")
	t.Setenv("NBU1_APIKEY", "secret-api-key-1234")

	tests := []struct {
		name      string
		input     string
		want      string
		wantErr   bool
		errSubstr string
	}{
		{
			name:  "no references — passthrough",
			input: "literal-value",
			want:  "literal-value",
		},
		{
			name:  "single env reference expanded",
			input: "${NBU1_HOSTNAME}",
			want:  "nbu-master.example.com",
		},
		{
			name:  "reference embedded in larger string",
			input: "https://${NBU1_HOSTNAME}:1556/netbackup",
			want:  "https://nbu-master.example.com:1556/netbackup",
		},
		{
			name:  "multiple env references in one string",
			input: "${NBU1_HOSTNAME}:${NBU1_APIKEY}",
			want:  "nbu-master.example.com:secret-api-key-1234",
		},
		{
			name:      "unset variable returns error",
			input:     "${UNSET_VAR_THAT_DOES_NOT_EXIST}",
			wantErr:   true,
			errSubstr: "UNSET_VAR_THAT_DOES_NOT_EXIST",
		},
		{
			name:      "multiple unset variables listed in error",
			input:     "${MISSING_A} and ${MISSING_B}",
			wantErr:   true,
			errSubstr: "MISSING_A",
		},
		{
			name:  "empty string — no error",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandEnv(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ExpandEnv(%q) = %q, nil; want error containing %q", tt.input, got, tt.errSubstr)
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ExpandEnv(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ExpandEnv(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
