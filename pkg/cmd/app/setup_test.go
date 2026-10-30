package app

import (
	"io"
	"strings"
	"testing"

	"github.com/triptechtravel/slackbuzz-cli/internal/config"
	"github.com/triptechtravel/slackbuzz-cli/internal/iostreams"
)

func TestConfigTokenFrom(t *testing.T) {
	const good = "xoxe.xoxp-1-abc"
	tests := []struct {
		name     string
		stdin    string
		src      tokenSource
		env      string
		saved    string
		want     string
		wantSave bool
		wantErr  string
	}{
		{name: "stdin wins over env and saved", stdin: good + "\n", src: tokenSource{withToken: true}, env: "xoxe.xoxp-env", saved: "xoxe.xoxp-saved", want: good, wantSave: true},
		{name: "env beats saved and isn't saved", env: "  xoxe.xoxp-env  ", saved: "xoxe.xoxp-saved", want: "xoxe.xoxp-env"},
		{name: "blank env falls through to saved", env: "   ", saved: "xoxe.xoxp-saved", want: "xoxe.xoxp-saved"},
		{name: "nothing available", wantErr: errNoConfigToken.Error()},
		{name: "refresh token rejected", stdin: "xoxe-1-refresh\n", src: tokenSource{withToken: true}, wantErr: "refresh token"},
		{name: "bot token rejected", env: "xoxb-123", wantErr: "isn't an App Configuration"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(configTokenEnv, tt.env)
			ios := iostreams.Test()
			ios.In = io.NopCloser(strings.NewReader(tt.stdin))

			got, save, err := configTokenFrom(ios, tt.src, &config.AuthConfig{ConfigToken: tt.saved})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want || save != tt.wantSave {
				t.Fatalf("got (%q, save=%v), want (%q, save=%v)", got, save, tt.want, tt.wantSave)
			}
		})
	}
}
