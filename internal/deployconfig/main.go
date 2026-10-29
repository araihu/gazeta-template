// This command connects native Margo identity to a Wrangler configuration.
// Editorial rendering and validation remain entirely in the Gazeta dependency.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/araihu/margo/site"
)

type deployment struct {
	Name              string  `json:"name"`
	CompatibilityDate string  `json:"compatibility_date"`
	WorkersDev        bool    `json:"workers_dev"`
	PreviewURLs       bool    `json:"preview_urls"`
	Routes            []route `json:"routes"`
	Assets            assets  `json:"assets"`
}
type route struct {
	Pattern      string `json:"pattern"`
	CustomDomain bool   `json:"custom_domain"`
}
type assets struct {
	Directory        string `json:"directory"`
	HTMLHandling     string `json:"html_handling"`
	NotFoundHandling string `json:"not_found_handling"`
}

var workerName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)

func configuration(origin, worker, assetsDirectory string, allowPlaceholder bool) (deployment, error) {
	if !workerName.MatchString(worker) {
		return deployment{}, fmt.Errorf("worker must contain 1-63 lowercase letters, digits or hyphens, starting with a letter")
	}
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Port() != "" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return deployment{}, fmt.Errorf("site.base_url must be an HTTPS origin without a port or path")
	}
	host := strings.ToLower(u.Hostname())
	if net.ParseIP(host) != nil || !strings.Contains(host, ".") {
		return deployment{}, fmt.Errorf("site.base_url requires a domain name")
	}
	if !allowPlaceholder {
		for _, reserved := range []string{"example", "test", "invalid", "localhost", "example.com", "example.net", "example.org"} {
			if host == reserved || strings.HasSuffix(host, "."+reserved) {
				return deployment{}, fmt.Errorf("replace placeholder site.base_url before deploying: %s", host)
			}
		}
	}
	return deployment{Name: worker, CompatibilityDate: "2026-10-29", Routes: []route{{Pattern: host, CustomDomain: true}}, Assets: assets{Directory: assetsDirectory, HTMLHandling: "auto-trailing-slash", NotFoundHandling: "404-page"}}, nil
}
func run() error {
	worker := flag.String("worker", "", "Cloudflare Worker name")
	output := flag.String("out", ".build/wrangler.json", "generated Wrangler config path")
	allow := flag.Bool("allow-placeholder", false, "allow example domains for a credentials-free Wrangler dry run")
	flag.Parse()
	config, err := site.LoadConfig("site.yaml")
	if err != nil {
		return err
	}
	public, err := filepath.Abs("public")
	if err != nil {
		return err
	}
	outputDirectory, err := filepath.Abs(filepath.Dir(*output))
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(outputDirectory, public)
	if err != nil {
		return err
	}
	d, err := configuration(config.Site.BaseURL, *worker, filepath.ToSlash(relative), *allow)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		return err
	}
	return os.WriteFile(*output, append(data, '\n'), 0644)
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
