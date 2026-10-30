package schema_test

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/araihu/gazeta"
	"github.com/pelletier/go-toml/v2"
	"go.yaml.in/yaml/v3"
	"golang.org/x/net/html"
)

type person struct {
	Type      string `json:"@type"`
	ID        string `json:"@id"`
	Name, URL string
}
type item struct {
	Type      string `json:"@type"`
	Position  int
	Name, URL string
}
type entity struct {
	Context                          string `json:"@context"`
	Type                             string `json:"@type"`
	ID                               string `json:"@id"`
	URL, Name, Headline, Description string
	Language                         string `json:"inLanguage"`
	Published                        string `json:"datePublished"`
	Modified                         string `json:"dateModified"`
	MainEntityOfPage                 string
	Author                           []person
	Image                            *string
	Graph                            []entity `json:"@graph"`
	MainEntity                       *entity
	IsPartOf                         struct {
		ID string `json:"@id"`
	}
	NumberOfItems   int
	Items           []item `json:"itemListElement"`
	PotentialAction json.RawMessage
}
type page struct {
	Schema      entity
	Raw         string
	Canonical   string
	Language    string
	Metadata    map[string][]string
	AuthorLinks []string
	Published   string
	Refs        []string
}

// Read the delivered HTML, independently of Gazeta's schema constructors/checkers.
func parsePage(data []byte, canonical string, needsSchema bool) (page, error) {
	p := page{Metadata: map[string][]string{}}
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return p, err
	}
	var schemas []string
	canonicalCount := 0
	misplaced := false
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, inHead bool) {
		if n.Type == html.ElementNode {
			attrs := map[string]string{}
			for _, a := range n.Attr {
				attrs[a.Key] = a.Val
			}
			inHead = inHead || n.Data == "head"
			if n.Data == "html" {
				p.Language = attrs["lang"]
			}
			if n.Data == "link" && attrs["rel"] == "canonical" {
				canonicalCount++
				p.Canonical = attrs["href"]
				misplaced = misplaced || !inHead
			}
			if n.Data == "meta" && inHead {
				key := attrs["property"]
				if key == "" {
					key = attrs["name"]
				}
				p.Metadata[key] = append(p.Metadata[key], attrs["content"])
			}
			if n.Data == "a" {
				p.Refs = append(p.Refs, attrs["href"])
				if attrs["rel"] == "author" {
					p.AuthorLinks = append(p.AuthorLinks, attrs["href"])
				}
			}
			if n.Data == "time" && attrs["data-margo-publication-date"] == "published" {
				p.Published = attrs["datetime"]
			}
			if n.Data == "script" && attrs["type"] == "application/ld+json" {
				var text strings.Builder
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.TextNode {
						text.WriteString(c.Data)
					}
				}
				schemas = append(schemas, text.String())
				misplaced = misplaced || !inHead
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inHead)
		}
	}
	walk(doc, false)
	if canonicalCount != 1 || p.Canonical != canonical || misplaced {
		return p, fmt.Errorf("canonical/head differs from published identity: %q", p.Canonical)
	}
	if p.Language != "pt-BR" {
		return p, fmt.Errorf("incorrect HTML language: %s", p.Language)
	}
	want := 0
	if needsSchema {
		want = 1
	}
	if len(schemas) != want {
		return p, fmt.Errorf("expected %d JSON-LD blocks in head, got %d", want, len(schemas))
	}
	if !needsSchema {
		return p, nil
	}
	p.Raw = schemas[0]
	if err := json.Unmarshal([]byte(p.Raw), &p.Schema); err != nil {
		return p, fmt.Errorf("invalid JSON-LD: %w", err)
	}
	if p.Schema.Context != "https://schema.org" {
		return p, fmt.Errorf("incorrect JSON-LD context")
	}
	return p, nil
}

type author struct{ Name, Profile string }
type identity struct {
	Name, Domain, Profile, Color string
	Authors                      []author
	Cover                        bool
	Modified                     string
}

// Portuguese fixture values exercise localized content and Unicode serialization.
var identities = []identity{
	{Name: "Notas da Ana", Domain: "https://ana.example", Profile: "/about/", Color: "#306a4b", Authors: []author{{Name: `Ana "A" Silva`, Profile: "/about/"}}},
	{Name: "Caderno do Bruno", Domain: "https://bruno.example", Profile: "/autores/bruno/", Color: "#864a28", Authors: []author{{Name: "Bruno Costa", Profile: "/autores/bruno/"}, {Name: "Carla Santos", Profile: "https://people.example/carla/"}, {Name: "Pessoa Convidada"}}, Cover: true, Modified: "2026-02-02T10:00:00-03:00"},
	{Name: "Diário da Lia", Domain: "https://lia.example", Profile: "/about/", Color: "#583e88", Authors: []author{{Name: "Lia Rocha", Profile: "/about/"}}},
}

const title = "Artigo de teste"
const published = "2026-01-01T09:00:00-03:00"
const postRoute = "/blogs/primeiro-post/"

func profileURL(id identity, a author) string {
	if strings.HasPrefix(a.Profile, "/") {
		return id.Domain + a.Profile
	}
	return a.Profile
}

func readFixture(t *testing.T, root, file string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func writeFixture(t *testing.T, root, file string, data []byte) {
	t.Helper()
	file = filepath.Join(root, file)
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, data, 0644); err != nil {
		t.Fatal(err)
	}
}
func writeYAML(t *testing.T, root, file string, v any) {
	t.Helper()
	b, err := yaml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, file, b)
}
func writeMarkdown(t *testing.T, root, file string, v map[string]any, body string) {
	t.Helper()
	b, err := toml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, file, []byte("+++\n"+string(b)+"+++\n"+body))
}

// Fixtures intentionally do not depend on the starter's demo post, author or assets.
// A generated blog can rename or remove those files without rewriting this suite.
func customize(t *testing.T, root string, id identity) {
	t.Helper()
	if err := os.RemoveAll(filepath.Join(root, "content")); err != nil {
		t.Fatal(err)
	}
	writeYAML(t, root, "site.yaml", map[string]any{
		"version": 1, "source": ".build/content", "output": "public", "assets": "local", "offline": true,
		"site":        map[string]any{"name": id.Name, "description": "Publicação editorial de teste.", "base_url": id.Domain, "home": "index.md", "logo": "assets/mark.svg", "icon": "assets/mark.svg", "social_image": map[string]string{"path": "assets/social.png", "alt": "Preview editorial de teste"}},
		"locales":     map[string]any{"default": "pt-BR", "supported": []string{"pt-BR"}},
		"publication": map[string]string{"date_format": "02/01/2006"},
		"layout":      map[string]any{"kind": "docs", "default": map[string]bool{"sidebar": false, "toc": false, "pagination": false}},
		"navigation":  map[string]any{"links": []map[string]string{{"label": "Blog", "href": "/blogs/"}, {"label": "Autor", "href": id.Profile}}},
		"theme":       map[string]any{"builtin": true, "name": "modern", "color_mode": "system", "allow_switch_theme": true},
		"custom_css":  []map[string]string{{"css_url": "assets/theme.css"}},
	})
	profiles := map[string]any{}
	for _, a := range id.Authors {
		if a.Profile != "" {
			profiles[a.Name] = map[string]string{"url": a.Profile}
		}
	}
	writeYAML(t, root, "blog.yaml", map[string]any{"version": 1, "home_limit": 6, "authors": profiles, "page_types": map[string]string{id.Profile: "ProfilePage"}})
	writeMarkdown(t, root, "content/index.md", map[string]any{"title": id.Name}, "Conheça [quem escreve]("+id.Profile+").\n")
	writeMarkdown(t, root, "content"+id.Profile+"index.md", map[string]any{"title": "Sobre " + id.Authors[0].Name, "description": "Bio de " + id.Authors[0].Name}, "Sou "+id.Authors[0].Name+".\n")
	names := []string{}
	for _, a := range id.Authors {
		names = append(names, a.Name)
	}
	post := map[string]any{"title": title, "description": "Conteúdo e configuração de uma publicação.", "publishedAt": published, "authors": names, "tags": []string{"Começo"}}
	if id.Modified != "" {
		post["modifiedAt"] = id.Modified
	}
	if id.Cover {
		post["cover"] = "./assets/capa.png"
		post["cover_alt"] = "Capa editorial de teste"
		writeImage(t, root, "content/blogs/primeiro-post/assets/capa.png", 640, 360)
	}
	writeMarkdown(t, root, "content/blogs/primeiro-post/index.md", post, "Este artigo pertence à identidade configurada.\n")
	writeFixture(t, root, "assets/mark.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><path d="M4 4h32v32H4z" fill="#306a4b"/></svg>`))
	writeImage(t, root, "assets/social.png", 1280, 640)
	writeFixture(t, root, "assets/theme.css", []byte("body { --color-primary: "+id.Color+"; }"))
}
func writeImage(t *testing.T, root, name string, width, height int) {
	t.Helper()
	file := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{40, 90, 70, 255}), image.Point{}, draw.Src)
	err = png.Encode(f, img)
	closeErr := f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
func graphEntity(graph []entity, kind string) (entity, error) {
	var found []entity
	for _, e := range graph {
		if e.Type == kind {
			found = append(found, e)
		}
	}
	if len(found) != 1 {
		return entity{}, fmt.Errorf("expected one %s in home graph", kind)
	}
	return found[0], nil
}
func listMatches(p page, e *entity, id identity) bool {
	return e != nil && e.Type == "ItemList" && e.NumberOfItems == 1 && len(e.Items) == 1 && e.Items[0].Type == "ListItem" && e.Items[0].Position == 1 && e.Items[0].Name == title && e.Items[0].URL == id.Domain+postRoute && (contains(p.Refs, postRoute) || contains(p.Refs, id.Domain+postRoute))
}

func audit(root string, id identity, forbidden []string) error {
	pages := map[string]page{}
	err := filepath.WalkDir(filepath.Join(root, "public"), func(file string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(file) != ".html" {
			return nil
		}
		relative, err := filepath.Rel(filepath.Join(root, "public"), file)
		if err != nil {
			return err
		}
		route := "/" + filepath.ToSlash(relative)
		if filepath.Base(file) == "index.html" {
			route = strings.TrimSuffix(route, "index.html")
		}
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		p, err := parsePage(b, id.Domain+route, route != "/404.html")
		if err != nil {
			return fmt.Errorf("%s: %w", route, err)
		}
		var decoded any
		if p.Raw != "" {
			if err := json.Unmarshal([]byte(p.Raw), &decoded); err != nil {
				return err
			}
		}
		if old := oldIdentity(decoded, forbidden); old != "" {
			return fmt.Errorf("%s: old identity %q in JSON-LD", route, old)
		}
		if route != "/404.html" && route != postRoute && p.Schema.Type == "BlogPosting" {
			return fmt.Errorf("article schema leaked to %s", route)
		}
		pages[route] = p
		return nil
	})
	if err != nil {
		return err
	}
	home := pages["/"]
	website, err := graphEntity(home.Schema.Graph, "WebSite")
	if err != nil {
		return err
	}
	if website.Name != id.Name || website.URL != id.Domain+"/" || website.ID != id.Domain+"/#website" || website.Language != "pt-BR" || len(website.PotentialAction) != 0 {
		return fmt.Errorf("incorrect WebSite identity or invented search action")
	}
	webpage, err := graphEntity(home.Schema.Graph, "WebPage")
	if err != nil {
		return err
	}
	if webpage.URL != home.Canonical || webpage.ID != home.Canonical+"#webpage" || webpage.Name != id.Name || webpage.IsPartOf.ID != website.ID || !listMatches(home, webpage.MainEntity, id) {
		return fmt.Errorf("home WebPage/list differs from visible identity")
	}
	article := pages[postRoute]
	n := article.Schema
	if n.Type != "BlogPosting" || n.URL != article.Canonical || n.ID != article.Canonical+"#blogposting" || n.MainEntityOfPage != article.Canonical || n.Headline != title || n.Description != "Conteúdo e configuração de uma publicação." || n.Language != "pt-BR" || n.Published != published || article.Published != n.Published || n.Modified != id.Modified {
		return fmt.Errorf("article identity, content or dates differ from source")
	}
	if len(n.Author) != len(id.Authors) {
		return fmt.Errorf("author count differs from source")
	}
	for i, a := range id.Authors {
		if n.Author[i].Type != "Person" || n.Author[i].Name != a.Name || n.Author[i].URL != profileURL(id, a) {
			return fmt.Errorf("incorrect author/profile at position %d", i)
		}
		if a.Profile != "" && !contains(article.AuthorLinks, profileURL(id, a)) {
			return fmt.Errorf("configured author profile missing from visible header")
		}
	}
	if id.Cover {
		want := id.Domain + postRoute + "assets/capa.png"
		if n.Image == nil || *n.Image != want {
			return fmt.Errorf("article image differs from cover")
		}
		for _, key := range []string{"og:image", "twitter:image"} {
			values := article.Metadata[key]
			if len(values) != 1 || values[0] != want {
				return fmt.Errorf("%s differs from structured cover", key)
			}
		}
	} else if n.Image != nil {
		return fmt.Errorf("article image invented without a cover")
	}
	profile := pages[id.Profile].Schema
	if profile.Type != "ProfilePage" || profile.URL != id.Domain+id.Profile || profile.ID != profile.URL+"#webpage" || profile.IsPartOf.ID != website.ID || profile.MainEntity == nil {
		return fmt.Errorf("incorrect ProfilePage identity")
	}
	person := profile.MainEntity
	if person.Type != "Person" || person.Name != id.Authors[0].Name || person.URL != profile.URL || person.ID != profile.URL+"#person" || person.Image != nil {
		return fmt.Errorf("profile mainEntity differs from configured author")
	}
	for _, route := range []string{"/blogs/", "/tags/comeco/"} {
		p := pages[route]
		if p.Schema.Type != "CollectionPage" || p.Schema.URL != p.Canonical || p.Schema.ID != p.Canonical+"#webpage" || p.Schema.IsPartOf.ID != website.ID || !listMatches(p, p.Schema.MainEntity, id) {
			return fmt.Errorf("incorrect collection identity/list: %s", route)
		}
	}
	return nil
}

func build(t *testing.T, root string) {
	t.Helper()
	if err := gazeta.Build(context.Background(), root, time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
}

func TestStructuredDataAcrossDomainAndAuthorChanges(t *testing.T) {
	root := t.TempDir()
	forbidden := []string{"https://blog.example", "Seu Nome", "Meu blog"}
	for _, id := range identities {
		t.Run(id.Name, func(t *testing.T) {
			customize(t, root, id)
			build(t, root)
			if err := audit(root, id, forbidden); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(readFixture(t, root, "public/assets/theme.css")), id.Color) {
				t.Fatal("custom theme lost")
			}
		})
		forbidden = append(forbidden, id.Domain, id.Name)
		for _, a := range id.Authors {
			forbidden = append(forbidden, a.Name)
		}
	}
}

func oldIdentity(value any, forbidden []string) string {
	switch v := value.(type) {
	case string:
		for _, old := range forbidden {
			if strings.Contains(v, old) {
				return old
			}
		}
	case []any:
		for _, item := range v {
			if old := oldIdentity(item, forbidden); old != "" {
				return old
			}
		}
	case map[string]any:
		for _, item := range v {
			if old := oldIdentity(item, forbidden); old != "" {
				return old
			}
		}
	}
	return ""
}

// Deliberately corrupt delivered metadata to prove the downstream contract fails,
// even if an upstream build/check were ever to accept the same regression.
func TestContractRejectsIdentityRegressions(t *testing.T) {
	root := t.TempDir()
	id := identities[1]
	customize(t, root, id)
	build(t, root)
	forbidden := []string{"https://blog.example", "Seu Nome", identities[0].Domain, identities[0].Authors[0].Name}
	if err := audit(root, id, forbidden); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, file string
		change     func(*testing.T, []byte) []byte
	}{
		{"stale article domain", "blogs/primeiro-post/index.html", func(t *testing.T, b []byte) []byte {
			return editJSON(t, b, func(n map[string]any) { n["url"] = identities[0].Domain + postRoute })
		}},
		{"missing author", "blogs/primeiro-post/index.html", func(t *testing.T, b []byte) []byte {
			return editJSON(t, b, func(n map[string]any) { delete(n, "author") })
		}},
		{"guest inherits owner profile", "blogs/primeiro-post/index.html", func(t *testing.T, b []byte) []byte {
			return editJSON(t, b, func(n map[string]any) { n["author"].([]any)[2].(map[string]any)["url"] = id.Domain + id.Profile })
		}},
		{"wrong profile person", "autores/bruno/index.html", func(t *testing.T, b []byte) []byte {
			return editJSON(t, b, func(n map[string]any) { n["mainEntity"].(map[string]any)["name"] = "Unrelated Person" })
		}},
		{"stale escaped name in nested data", "blogs/primeiro-post/index.html", func(t *testing.T, b []byte) []byte {
			return editJSON(t, b, func(n map[string]any) { n["citation"] = map[string]any{"name": identities[0].Authors[0].Name} })
		}},
		{"logo replaces article cover", "blogs/primeiro-post/index.html", func(t *testing.T, b []byte) []byte {
			return editJSON(t, b, func(n map[string]any) { n["image"] = id.Domain + "/assets/mark.svg" })
		}},
		{"duplicate JSON-LD", "blogs/primeiro-post/index.html", func(t *testing.T, b []byte) []byte {
			return []byte(strings.Replace(string(b), "</head>", `<script type="application/ld+json">{}</script></head>`, 1))
		}},
		{"wrong canonical", "blogs/primeiro-post/index.html", func(t *testing.T, b []byte) []byte {
			return []byte(strings.Replace(string(b), `rel="canonical" href="`+id.Domain+postRoute+`"`, `rel="canonical" href="`+id.Domain+`/other/"`, 1))
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name := "public/" + tc.file
			original := readFixture(t, root, name)
			changed := tc.change(t, original)
			if string(original) == string(changed) {
				t.Fatal("mutation did not change fixture")
			}
			writeFixture(t, root, name, changed)
			t.Cleanup(func() { writeFixture(t, root, name, original) })
			if err := audit(root, id, forbidden); err == nil {
				t.Fatal("corrupted metadata passed schema contract")
			}
		})
	}
}
func editJSON(t *testing.T, data []byte, edit func(map[string]any)) []byte {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	var raw string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, a := range n.Attr {
				if a.Key == "type" && a.Val == "application/ld+json" && n.FirstChild != nil {
					raw = n.FirstChild.Data
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if raw == "" {
		t.Fatal("fixture has no JSON-LD")
	}
	var node map[string]any
	if err := json.Unmarshal([]byte(raw), &node); err != nil {
		t.Fatal(err)
	}
	edit(node)
	replacement, err := json.Marshal(node)
	if err != nil {
		t.Fatal(err)
	}
	return []byte(strings.Replace(string(data), raw, string(replacement), 1))
}
