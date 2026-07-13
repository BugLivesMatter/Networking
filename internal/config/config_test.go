package config

import "testing"

func TestParseCSVOrigins(t *testing.T) {
	got := parseCSV(" http://localhost:3000, https://buglivesmatter.github.io, ,http://localhost:3000,*")
	want := []string{"http://localhost:3000", "https://buglivesmatter.github.io"}
	if len(got) != len(want) {
		t.Fatalf("origins = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("origin[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestValidateClusterRejectsUnknownSource(t *testing.T) {
	cfg := &Config{ClusterSource: "kubernetes"}
	if err := cfg.validateCluster(); err == nil {
		t.Fatal("validateCluster() error = nil, want unknown-source error")
	}
}

func TestValidateClusterDefaultsToDemo(t *testing.T) {
	cfg := &Config{ClusterSource: "demo"}
	if err := cfg.validateCluster(); err != nil {
		t.Fatalf("validateCluster() error = %v", err)
	}
}
