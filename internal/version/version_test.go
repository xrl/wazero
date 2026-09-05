package version

import (
	"runtime/debug"
	"testing"
)

func TestBuildInfoVersion(t *testing.T) {
	for _, tc := range []struct {
		name string
		dep  *debug.Module
		want string
	}{
		{"upstream", &debug.Module{Version: "v1.12.0"}, "v1.12.0"},
		{"fork tag", &debug.Module{Version: "v1.12.0", Replace: &debug.Module{Path: "github.com/xrl/wazero", Version: "v1.12.0-xrl.0"}}, "v1.12.0-xrl.0"},
		{"fork pseudoversion", &debug.Module{Version: "v1.12.0", Replace: &debug.Module{Path: "github.com/xrl/wazero", Version: "v1.12.1-0.20260701000000-123456abcdef"}}, "v1.12.1-0.20260701000000-123456abcdef"},
		{"local replacement", &debug.Module{Version: "v1.12.0", Replace: &debug.Module{Path: "../wazero"}}, Default},
		{"devel replacement", &debug.Module{Version: "v1.12.0", Replace: &debug.Module{Version: "(devel)"}}, Default},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.dep.Path = "github.com/tetratelabs/wazero"
			info := &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}, Deps: []*debug.Module{tc.dep}}
			if got := buildInfoVersion(info); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
	for _, v := range []string{"v1.12.0-xrl.0", "", "(devel)"} {
		want := v
		if versionMissing(v) {
			want = Default
		}
		if got := buildInfoVersion(&debug.BuildInfo{Main: debug.Module{Version: v}}); got != want {
			t.Fatalf("main got %q, want %q", got, want)
		}
	}
}
