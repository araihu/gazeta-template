package schema_test

import (
	"context"
	"encoding/xml"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/araihu/gazeta"
	"go.yaml.in/yaml/v3"
)

func TestMultilingualOrganizationIdentities(t *testing.T) {
	root := t.TempDir()
	for _, id := range []identity{
		{Name: "Workshop Notes", Domain: "https://workshop.example", Profile: "/about/", Color: "#173b72", Authors: []author{{Name: "Workshop Collective", Profile: "/about/"}}},
		{Name: "Garden Journal", Domain: "https://garden.example", Profile: "/organization/", Color: "#306a4b", Authors: []author{{Name: "Garden Collective", Profile: "/organization/"}}},
	} {
		customize(t, root, id)
		var site map[string]any
		if err := yaml.Unmarshal(readFixture(t, root, "site.yaml"), &site); err != nil {
			t.Fatal(err)
		}
		site["locales"] = map[string]any{"default": "en", "supported": []string{"en", "pt-BR", "es"}}
		site["navigation"] = map[string]any{}
		writeYAML(t, root, "site.yaml", site)
		writeYAML(t, root, "blog.yaml", map[string]any{"version": 1, "authors": map[string]any{id.Authors[0].Name: map[string]string{"url": id.Profile, "type": "Organization"}}, "page_types": map[string]string{id.Profile: "AboutPage"}})
		for locale, prefix := range map[string]string{"en": "", "pt-BR": "pt-br/", "es": "es/"} {
			writeMarkdown(t, root, "content/"+prefix+"index.md", map[string]any{"title": id.Name, "language": locale}, id.Name+".\n")
			writeMarkdown(t, root, "content/"+prefix+strings.TrimPrefix(id.Profile, "/")+"index.md", map[string]any{"title": id.Authors[0].Name, "language": locale}, id.Authors[0].Name+".\n")
			writeMarkdown(t, root, "content/"+prefix+"blogs/primeiro-post/index.md", map[string]any{"title": title, "description": "Independent localized fixture.", "language": locale, "authors": []string{id.Authors[0].Name}, "publishedAt": published}, "A publication fixture.\n")
		}
		now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
		if err := gazeta.Build(context.Background(), root, now); err != nil {
			t.Fatal(err)
		}
		if err := gazeta.Check(root, now); err != nil {
			t.Fatal(err)
		}
		for locale, prefix := range map[string]string{"en": "", "pt-BR": "pt-br/", "es": "es/"} {
			route := "/" + prefix + "blogs/primeiro-post/"
			p, err := parsePage(readFixture(t, root, "public/"+prefix+"blogs/primeiro-post/index.html"), id.Domain+route, true, locale)
			if err != nil {
				t.Fatal(err)
			}
			if p.Schema.Type != "BlogPosting" || p.Schema.Language != locale || p.Schema.URL != id.Domain+route || len(p.Schema.Author) != 1 || p.Schema.Author[0].Type != "Organization" || p.Schema.Author[0].Name != id.Authors[0].Name || p.Schema.Author[0].URL != id.Domain+"/"+prefix+strings.TrimPrefix(id.Profile, "/") {
				t.Fatalf("incorrect institutional article: %+v", p.Schema)
			}
			if len(p.Alternates) != 4 || len(p.Alternates[locale]) != 1 || p.Alternates[locale][0] != p.Canonical || len(p.Alternates["x-default"]) != 1 || p.Alternates["x-default"][0] != id.Domain+postRoute {
				t.Fatalf("incorrect equivalents: %+v", p.Alternates)
			}
			profile := "/" + prefix + strings.TrimPrefix(id.Profile, "/")
			about, err := parsePage(readFixture(t, root, filepath.Join("public", prefix, id.Profile, "index.html")), id.Domain+profile, true, locale)
			if err != nil {
				t.Fatal(err)
			}
			if about.Schema.Type != "AboutPage" || about.Schema.MainEntity == nil || about.Schema.MainEntity.Type != "Organization" {
				t.Fatal("institution became a Person profile")
			}
			var feed struct {
				Channel struct {
					Language string `xml:"language"`
					Items    []struct {
						Link string `xml:"link"`
					} `xml:"item"`
				} `xml:"channel"`
			}
			if err := xml.Unmarshal(readFixture(t, root, "public/"+prefix+"index.xml"), &feed); err != nil {
				t.Fatal(err)
			}
			if feed.Channel.Language != locale || len(feed.Channel.Items) != 1 || feed.Channel.Items[0].Link != id.Domain+route {
				t.Fatal("feed mixes locale identities")
			}
			if id.Name == "Garden Journal" && strings.Contains(p.Raw, "workshop.example") {
				t.Fatal("previous identity leaked")
			}
		}
	}
}
