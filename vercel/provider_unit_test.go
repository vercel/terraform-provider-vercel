package vercel

import "testing"

func TestAPITokenRegexMatchesWholeToken(t *testing.T) {
	for _, tt := range []struct {
		name  string
		token string
		want  bool
	}{
		{
			name:  "valid token",
			token: "abcDEF123456abcDEF123456",
			want:  true,
		},
		{
			name:  "too long with valid substring",
			token: "abcDEF123456abcDEF123456!",
			want:  false,
		},
		{
			name:  "invalid character",
			token: "abcDEF123456abcDEF12345!",
			want:  false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := apiTokenRe.MatchString(tt.token); got != tt.want {
				t.Fatalf("apiTokenRe.MatchString(%q) = %t, want %t", tt.token, got, tt.want)
			}
		})
	}
}
