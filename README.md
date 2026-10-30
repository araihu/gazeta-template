# Gazeta blog template

A Git-managed blog using [Gazeta v0.0.4](https://github.com/araihu/gazeta/releases/tag/v0.0.4), native Margo configuration and Cloudflare Workers Static Assets.

The engine is a Go tool pinned in `go.mod`. This repository contains configuration, sample content, original assets and a small adapter between the site's identity and deployment. There is no engine copy to maintain.

## Create your blog

1. Select **Use this template > Create a new repository**. Include the `main` branch.
2. Clone your new repository. Local requirements are Go 1.27.3+, Node.js 24+ and Make.
3. Edit the files below and replace the sample content.
4. Run `npm ci` and `make check dry-run`. Open a PR to review the first publication.
5. Configure your own deployment credentials and enable deployment as described below.

| Setting | File |
| --- | --- |
| Name, description and canonical domain | `site.yaml`, `site.*` fields |
| Authors and profile destinations | `blog.yaml`, `authors` field |
| Author page type | `blog.yaml`, `page_types` field |
| Biography and article authorship | `content/about/` and post frontmatter |
| Theme, light/dark mode and theme toggle | `site.yaml`, `theme` field |
| Palette and listing layout | `assets/theme.css` |
| Logo, icon and 1280x640 social preview | `assets/logo.svg`, `assets/mark.svg` and `assets/social.png`, referenced in `site.yaml` |

Replace `https://blog.example` with your real domain, without a subdirectory. Deployment derives the Custom Domain from that same field, so there is no second domain value to synchronize. Replace `Seu Nome` in the author registry, biography and article frontmatter. Guest authors can have their own profiles and pages.

The theme uses public Margo/Goshtoso tokens. Gazeta supplies the editorial `.blog-*` classes. Changing colors, logos or CSS layout does not require Go changes. Gazeta supports one language per blog. If you change the language, update `locales` and the editorial labels in `blog.yaml`.

## Configure CI/CD

The destination is a Worker serving `public/` as Static Assets, with directory URLs and a custom 404 page. The domain must belong to an active zone in your Cloudflare account and be available to attach to the Worker as a Custom Domain.

In your new repository's **Settings > Secrets and variables > Actions**, configure:

| Type | Name | Value |
| --- | --- | --- |
| Variable | `CLOUDFLARE_ACCOUNT_ID` | ID of the account hosting this blog |
| Variable | `CLOUDFLARE_WORKER_NAME` | Unique Worker name, such as `my-blog` |
| Variable | `CLOUDFLARE_DEPLOY_ENABLED` | `true`, after configuring the domain and credentials |
| Secret | `CLOUDFLARE_API_TOKEN` | Cloudflare token allowed to edit Workers and attach the domain in the selected account/zone |

Use the **Edit Cloudflare Workers** token template, restricted to this blog's account and zone. Follow the official [Cloudflare GitHub Actions instructions](https://developers.cloudflare.com/workers/ci-cd/external-cicd/github-actions/) and [GitHub secrets documentation](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-secrets). Never put the token in tracked files or commands.

This template does not set those variables or include credentials. Automatic deployment stays disabled until the new repository is configured. The deployment generator also rejects placeholder domains for real publication.

### Workflow

- **Pull request:** Gazeta build and validation, configuration and cross-identity JSON-LD tests, `go vet`, Wrangler dry-run and an artifact containing the generated site. No deployment secrets are used.
- **Push to `main`:** when deployment is enabled, a separate workflow prepares and validates the publication. A second job downloads that run's artifact and publishes those exact files, without rebuilding the site with production credentials.
- **Manual run:** in Actions > Deploy blog > Run workflow, keep `dry_run = true` to test the complete pipeline without publishing. This works in the template itself, without an account or token. To publish manually, select `main`, configure the variables above and clear `dry_run`.

After a real deployment, Gazeta verifies the public origin against the downloaded artifact without rebuilding. It checks Margo manifest identity, HTML/JSON-LD/canonical metadata, asset bytes and MIME types, legacy HTTP redirects and missing-page status. A three-minute propagation deadline bounds retries. Failure makes the deployment job fail; `production-verification-<commit>` preserves the JSON receipt. Run `make verify` against an existing local `public/` directory to repeat this check. The verifier does not replace Google's external Rich Results Test.

Deployments run serially. Before publishing, the workflow checks that its commit is still the current `main`; an older run cannot replace a newer publication. Only the real deployment step receives the Cloudflare token. Do not run another automatic deployment pipeline for the same Worker at the same time.

Wrangler is pinned in `package-lock.json`. `.build/wrangler.json` is generated from the site configuration and must not be edited or committed. The pipeline also runs a dry-run against the downloaded artifact to verify paths between jobs.

## Development

```sh
npm ci
make check dry-run  # Build, validate, test and package without publishing
make run            # Preview at http://127.0.0.1:1313
```

Rebuild after editing. Node/Wrangler are needed only for packaging and deployment; `make build` and the preview use Go/Gazeta.

### Structured data contract

`make check` runs the `internal/schema` suite without cached test results, both in PR CI and before deployment. It builds three successive identities in the same temporary directory, with different domains, authors, profile paths and themes. Fixtures are independent of the starter content, so you can remove the sample post without rewriting the tests.

The suite reads the final HTML, decodes JSON-LD and checks:

- `WebSite`, the home page, `BlogPosting`, `ProfilePage`/`Person` and article lists use the current identity's URLs, IDs and canonical links.
- Single or multiple authors, local or external profiles and a guest without a URL. The profile must match the configured person.
- Publication and modification dates, articles with and without covers, and matching cover URLs in Open Graph/Twitter metadata. Logos and the global preview do not become article or person images.
- Exactly one JSON-LD block in the head of each regular page, none on the 404 page, and no previous identity in decoded JSON values, including escaped text.

The test also corrupts eight valid outputs to prove that the contract rejects regressions. To run only this suite:

```sh
GOWORK=off go test ./internal/schema -count=1 -v
```

Gazeta also requires exact sitemap coverage of indexable canonical pages. The generated 404 page uses Margo's native `noindex, follow` metadata and is omitted from the sitemap and `llms.txt`. Aliases retain their redirect targets instead of separate sitemap entries.

This local validation checks the publication contract without calling Google services from CI. After a real deployment, validate the URLs in the [Rich Results Test](https://search.google.com/test/rich-results) and [Schema.org Validator](https://validator.schema.org/). Eligibility rules are documented under [Article](https://developers.google.com/search/docs/appearance/structured-data/article), [ProfilePage](https://developers.google.com/search/docs/appearance/structured-data/profile-page) and [site names](https://developers.google.com/search/docs/appearance/site-names).

## Publish content

Create `content/blogs/my-article/index.md`:

```toml
+++
title = "Article title"
description = "A summary of what the reader will find."
authors = ["Seu Nome"]
publishedAt = "2026-10-29T10:00:00-03:00"
tags = ["Topic"]
draft = true
+++
```

The directory defines `/blogs/my-article/`. Before publishing, review the text, provide the real publication date and set `draft` to `false`. Use RFC 3339 timestamps with a timezone. Future posts and drafts stay out of the publication, including their assets. A future post needs another build after its publication time; push a commit or trigger a manual run, since this template does not create a schedule.

Keep article images in the article's `assets/` directory. Set `cover = "./assets/cover.png"` and `cover_alt` when a real cover exists. The global social preview is not used as the article image in JSON-LD. Series use `series` and a positive, unique `seriesPart`.

Gazeta/Margo own authorship, dates, covers, tags, RSS, archives and JSON-LD. A biography configured as `ProfilePage` identifies one person; a general team page can use `AboutPage`. Authored HTML/JavaScript is disabled by default in `blog.yaml`.

After the first deployment, check the domain, RSS and author profile. Validate an article and the biography in the [Rich Results Test](https://search.google.com/test/rich-results). Passing does not guarantee indexing or rich-result display on Google.

## Updates and rollback

Update the engine in a PR with `go get -tool github.com/araihu/gazeta/cmd/gazeta@VERSION`, followed by `go mod tidy` and `make check dry-run`. The workflow and local commands use the same pinned version. Update Wrangler to an exact version and review the lockfile in the same PR.

To undo a publication, revert the commit and push a new commit to `main`. The workflow rebuilds the publication. In an emergency, you can also restore a previous version through Workers Deployments; then align Git so the next deployment does not restore the unwanted change.

## Repository language

Write repository documentation, GoDoc, source-code identifiers, code comments and developer-facing tooling messages in English, including upstream contributions. Blog content, published UI labels and author identities follow the configured locale. The starter blog and intentional localization/Unicode test fixtures use Brazilian Portuguese; preserve those values when they are needed to test localized output.

This template uses the MIT license. Dependencies retain their own licenses.
