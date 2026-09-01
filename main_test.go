package main

import "testing"

func TestEnvOr(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		envVal string
		envSet bool
		def    string
		want   string
	}{
		{name: "env set returns env value", key: "GMF_TEST_VAR", envVal: "from-env", envSet: true, def: "def", want: "from-env"},
		{name: "env unset returns default", key: "GMF_TEST_VAR", envSet: false, def: "def", want: "def"},
		{name: "env empty returns default", key: "GMF_TEST_VAR", envVal: "", envSet: true, def: "def", want: "def"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envSet {
				t.Setenv(tt.key, tt.envVal)
			}
			if got := envOr(tt.key, tt.def); got != tt.want {
				t.Errorf("envOr(%q, %q) = %q, want %q", tt.key, tt.def, got, tt.want)
			}
		})
	}
}
