package main

import "testing"

func TestDeploymentTarget(t *testing.T) {
	d, err := configuration("https://journal.araihu.com", "new-journal", "../public", false)
	if err != nil {
		t.Fatal(err)
	}
	if d.Routes[0].Pattern != "journal.araihu.com" || !d.Routes[0].CustomDomain || d.Assets.Directory != "../public" || d.WorkersDev || d.PreviewURLs {
		t.Fatalf("incorrect deployment: %+v", d)
	}
	for _, origin := range []string{"http://journal.araihu.com", "https://blog.example", "https://example.com", "https://a.example.org", "https://localhost", "https://127.0.0.1", "https://journal.araihu.com/path", "https://journal.araihu.com:443", "https://user@journal.araihu.com", "https://journal.araihu.com?q=1"} {
		if _, err := configuration(origin, "new-journal", "../public", false); err == nil {
			t.Errorf("accepted invalid production origin %s", origin)
		}
	}
	for _, name := range []string{"", "../worker", "My Worker", "a;exit 1"} {
		if _, err := configuration("https://journal.araihu.com", name, "../public", false); err == nil {
			t.Errorf("accepted worker %q", name)
		}
	}
	if _, err := configuration("https://blog.example", "gazeta-check", "../public", true); err != nil {
		t.Fatal("placeholder must work only in dry-run generation:", err)
	}
}
