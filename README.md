# Gazeta blog template

Um blog mantido em Git, com [Gazeta v0.0.2](https://github.com/araihu/gazeta/releases/tag/v0.0.2), configuração nativa do Margo e publicação no Cloudflare Workers Static Assets.

A engine é uma ferramenta Go versionada em `go.mod`. Este repositório contém configuração, conteúdo de exemplo, assets originais e uma pequena ligação entre a identidade do site e o deploy. Não é preciso manter uma cópia da engine.

## Criar seu blog

1. Clique em **Use this template > Create a new repository**. Inclua a branch `main`.
2. Clone o novo repositório. Requisitos locais: Go 1.27.3+, Node.js 24+ e Make.
3. Edite os arquivos abaixo e substitua o conteúdo de exemplo.
4. Execute `npm ci` e `make check dry-run`. Abra um PR para revisar a primeira publicação.
5. Configure as credenciais do seu próprio destino e habilite o deploy conforme a seção seguinte.

| Configuração | Arquivo |
| --- | --- |
| Nome, descrição e domínio canônico | `site.yaml`, campos `site.*` |
| Autores e destinos dos perfis | `blog.yaml`, campo `authors` |
| Tipo de página do autor | `blog.yaml`, campo `page_types` |
| Conteúdo da bio e autoria de cada artigo | `content/about/` e frontmatter dos posts |
| Tema, modo claro/escuro e alternância | `site.yaml`, campo `theme` |
| Paleta e disposição das listas | `assets/theme.css` |
| Ícone/logo e preview social de 1280×640 | `assets/mark.svg` e `assets/social.png`, referenciados em `site.yaml` |

Troque `https://blog.example` pelo domínio real, sem subdiretório. O deploy deriva o Custom Domain desse mesmo campo; não há outra cópia do domínio para sincronizar. Substitua `Seu Nome` tanto no cadastro de autores quanto na bio e no frontmatter do artigo. Autores convidados podem ter seus próprios perfis e páginas.

O tema usa os tokens públicos do Margo/Goshtoso. As classes editoriais `.blog-*` vêm da Gazeta. Trocar cores, logo ou layout CSS não exige modificar Go. A Gazeta suporta um idioma por blog; ajuste `locales` e os textos editoriais de `blog.yaml` se mudar de idioma.

## Configurar CI/CD

O destino é um Worker que serve `public/` como Static Assets, com URLs de diretório e página 404. O domínio precisa estar em uma zona ativa da sua conta Cloudflare e disponível para ser associado ao Worker como Custom Domain.

Em **Settings > Secrets and variables > Actions** do novo repositório, configure:

| Tipo | Nome | Valor |
| --- | --- | --- |
| Variable | `CLOUDFLARE_ACCOUNT_ID` | ID da conta que hospedará este blog |
| Variable | `CLOUDFLARE_WORKER_NAME` | Nome exclusivo do Worker, como `meu-blog` |
| Variable | `CLOUDFLARE_DEPLOY_ENABLED` | `true`, após configurar domínio e credenciais |
| Secret | `CLOUDFLARE_API_TOKEN` | Token Cloudflare para editar Workers e associar o domínio na conta/zona escolhida |

Use um token com o modelo **Edit Cloudflare Workers**, limitado à conta e zona deste blog. Veja as instruções oficiais de [Cloudflare com GitHub Actions](https://developers.cloudflare.com/workers/ci-cd/external-cicd/github-actions/) e [secrets no GitHub](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-secrets). Não coloque o token em arquivos ou comandos versionados.

O template não define essas variáveis nem inclui credenciais. O deploy automático fica desabilitado até a configuração do novo repositório. O gerador também recusa domínios de exemplo na publicação real.

### Fluxo

- **Pull request:** build e validação da Gazeta, testes de configuração/identidade, `go vet`, dry-run do Wrangler e artifact com o site gerado. Nenhum secret de deploy é usado.
- **Push em `main`:** com deploy habilitado, outro workflow prepara e valida a publicação. Um segundo job baixa o artifact daquela execução e publica esses mesmos arquivos, sem reconstruir o site com as credenciais de produção.
- **Execução manual:** em Actions > Deploy blog > Run workflow, mantenha `dry_run = true` para testar o pipeline completo sem publicar. Esse modo funciona até no template, sem conta ou token. Para publicar manualmente, use `main`, configure as variáveis acima e desmarque `dry_run`.

Deploys são serializados. Antes de publicar, o workflow confirma que o commit ainda é o `main` atual; uma execução antiga não substitui uma publicação mais recente. O token Cloudflare só é disponibilizado à etapa de deploy real. Não use simultaneamente outro pipeline de publicação automática para o mesmo Worker.

O Wrangler fica fixado em `package-lock.json`. `.build/wrangler.json` é gerado da configuração do site e não deve ser editado ou commitado. O pipeline gera também um dry-run a partir do artifact baixado, para conferir que os caminhos continuam válidos entre jobs.

## Desenvolvimento

```sh
npm ci
make check dry-run  # Gera, valida, testa e empacota sem publicar
make run            # Preview em http://127.0.0.1:1313
```

Refaça o build após editar. Node/Wrangler só são necessários para o empacotamento e deploy; `make build` e o preview usam Go/Gazeta.

`make check` também constrói uma segunda identidade em diretório temporário para conferir domínio, autor, perfil e tema independentes. `scripts/check.sh` contém as verificações Go que podem ser executadas novamente pelo revisor.

## Publicar conteúdo

Crie `content/blogs/meu-artigo/index.md`:

```toml
+++
title = "Título do artigo"
description = "Um resumo do que o leitor encontrará."
authors = ["Seu Nome"]
publishedAt = "2026-10-29T10:00:00-03:00"
tags = ["Assunto"]
draft = true
+++
```

O diretório define `/blogs/meu-artigo/`. Para publicar, revise o texto, informe a data real e altere `draft` para `false`. Use timestamps RFC 3339 com fuso. Datas futuras e rascunhos ficam fora da publicação, incluindo seus assets. Um post futuro precisa de outro build após a data; faça um push ou uma execução manual, pois este template não cria agendamento.

Imagens do artigo ficam em sua pasta `assets/`. Use `cover = "./assets/capa.png"` e `cover_alt` quando houver uma capa real. O preview global do site não é usado como imagem do artigo no JSON-LD. Séries usam `series` e um `seriesPart` positivo e único.

Autoria, datas, capas, tags, RSS, arquivos e JSON-LD são responsabilidade da Gazeta/Margo. A bio configurada como `ProfilePage` identifica uma pessoa; uma página geral sobre uma equipe pode usar `AboutPage`. O HTML/JavaScript autoral fica desabilitado por padrão em `blog.yaml`.

Após o primeiro deploy, confira o domínio, o RSS e o perfil do autor. Valide um artigo e a bio no [Rich Results Test](https://search.google.com/test/rich-results). O teste não garante indexação nem exibição em destaque no Google.

## Atualizações e rollback

Atualize a engine em um PR com `go get -tool github.com/araihu/gazeta/cmd/gazeta@VERSAO`, seguido de `go mod tidy` e `make check dry-run`. O workflow e os comandos locais usam a mesma versão fixada. Atualize o Wrangler com versão exata e revise o lockfile no mesmo PR.

Para desfazer uma publicação, reverta o commit e envie um novo commit a `main`. O workflow gera a publicação novamente. Em uma emergência, uma versão anterior também pode ser restaurada em Workers Deployments; alinhe o Git depois para que o próximo deploy não restaure a alteração indesejada.

O template tem licença MIT. As dependências mantêm suas próprias licenças.
