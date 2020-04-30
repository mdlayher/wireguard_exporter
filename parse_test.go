package wireguardexporter_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	wireguardexporter "github.com/mdlayher/wireguard_exporter"
)

func TestParsePeers(t *testing.T) {
	tests := []struct {
		name  string
		s     string
		peers map[string]string
		ok    bool
	}{
		{
			name: "bad TOML",
			s:    "xxx",
		},
		{
			name: "bad keys",
			s: `
			[bad]
			[[bad.bad]]
			`,
		},
		{
			name: "bad public key",
			s: `
			[[peer]]
			public_key = "x"
			`,
		},
		{
			name: "empty name",
			s: `
			[[peer]]
			public_key = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
			name = ""
			`,
		},
		{
			name: "ok",
			s: `
			[[peer]]
			public_key = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
			name = "foo"

			[[peer]]
			public_key = "AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI="
			name = "bar"
			`,
			peers: map[string]string{
				"AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=": "foo",
				"AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=": "bar",
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peers, err := wireguardexporter.ParsePeers(strings.NewReader(tt.s))
			if tt.ok && err != nil {
				t.Fatalf("failed to parse peer mappings: %v", err)
			}
			if !tt.ok && err == nil {
				t.Fatal("expected an error, but none occurred")
			}
			if err != nil {
				t.Logf("err: %v", err)
				return
			}

			if diff := cmp.Diff(tt.peers, peers); diff != "" {
				t.Fatalf("unexpected peers (-want +got):\n%s", diff)
			}
		})
	}
}
