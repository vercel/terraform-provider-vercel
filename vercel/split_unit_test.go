package vercel

import "testing"

func TestImportIDsRejectEmptyComponents(t *testing.T) {
	for _, parser := range []struct {
		name  string
		parse func(string) bool
		valid string
	}{
		{"two", func(id string) bool { _, _, ok := splitInto2(id); return ok }, "team/user"},
		{"one or two", func(id string) bool { _, _, ok := splitInto1Or2(id); return ok }, "project"},
		{"two or three", func(id string) bool { _, _, _, ok := splitInto2Or3(id); return ok }, "project/resource"},
		{"three or four", func(id string) bool { _, _, _, _, ok := splitInto3Or4(id); return ok }, "project/repository/grantee"},
	} {
		t.Run(parser.name, func(t *testing.T) {
			for _, id := range []string{"", "/", "/user", "team/", "/project/resource", "team//resource", "project/resource/", "team/project//grantee"} {
				if parser.parse(id) {
					t.Errorf("accepted import ID %q with an empty component", id)
				}
			}
			if !parser.parse(parser.valid) {
				t.Errorf("rejected valid ID %q", parser.valid)
			}
		})
	}
}
