package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/araihu/gazeta"
)

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

func TestIndependentBlogIdentity(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, dir := range []string{"content", "assets"} {
		if err := os.CopyFS(filepath.Join(root, dir), os.DirFS(filepath.Join(repo, dir))); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"site.yaml", "blog.yaml"} {
		data, err := os.ReadFile(filepath.Join(repo, file))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, file), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	err = filepath.WalkDir(root, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		switch filepath.Ext(file) {
		case ".md", ".yaml", ".css":
		default:
			return nil
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		changed := strings.NewReplacer("Meu blog", "Notas da Ana", "Seu Nome", "Ana Example", "https://blog.example", "https://ana.example", "#285b86", "#306a4b").Replace(string(data))
		return os.WriteFile(file, []byte(changed), 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := gazeta.Build(context.Background(), root, time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, "public", name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	article := read("blogs/primeiro-post/index.html")
	for _, want := range []string{"https://ana.example/blogs/primeiro-post/", "Ana Example", "https://ana.example/about/", "BlogPosting"} {
		if !strings.Contains(article, want) {
			t.Fatalf("missing identity %q", want)
		}
	}
	if strings.Contains(article, "Seu Nome") || strings.Contains(article, "https://blog.example") {
		t.Fatal("default identity leaked")
	}
	if !strings.Contains(read("about/index.html"), `"@type":"ProfilePage"`) {
		t.Fatal("author profile lost")
	}
	if !strings.Contains(read("assets/theme.css"), "#306a4b") {
		t.Fatal("custom theme lost")
	}
	var m struct{ Entries []struct{ Path string } }
	if err := json.Unmarshal([]byte(read("margo-manifest.json")), &m); err != nil || len(m.Entries) == 0 {
		t.Fatal("publication manifest missing", err)
	}
}
