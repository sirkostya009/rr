# rr directive highlighting

Syntax highlighting for rr `//rr:` comment directives in Go source.

It's an **injection** grammar — it colors tokens *inside* Go comments, so
normal Go highlighting is untouched. Nothing else in your `.go` files changes.

## What it colors

Line directives:

```go
//rr:api onerror=@handleError on405=@on405 on404=notFound on400=@badRequest
//rr:controller onerror=@postNotFound
//rr:pre @requireToken
//rr:route GET /api/posts/{slug=@isSlug} -- func checker takes what Atoi rejected
//rr:route GET /api/files/{path...}
```

Inline directives:

```go
func (u *UsersApi) Create( /* rr:body */ u User) {}
func (s *SearchApi) Search( /* rr:query @parseSearch */ q SearchQuery, /* rr:header X-Request-Id */ rid string) {}
```

Token scopes:

| token | scope |
| --- | --- |
| `rr:` prefix | `comment.directive.rr` |
| directive name (`api`, `controller`, `pre`, `route`, `body`, …) | `comment.directive.name.rr` |
| HTTP method | `support.constant.http-method.rr` |
| `key=` (onerror, on400, on404, on405) | `variable.parameter.config.rr` |
| `@ref` | `entity.name.function.reference.rr` |
| `{name}` / `...` inside a path param | `constant.character.escape.rr` (same as `\d` in a Go string) |
| `/path/segments` | `string.quoted.double.rr` (same as a Go string) |
| `*` catch-all | `keyword.operator.wildcard.rr` |
| `-- comment` suffix | `comment.line.double-dash.rr` |

The `rr:` prefix and directive name deliberately sit in the `comment.*` scope
so they read as muted by default — the meaningful tokens (path, method, refs)
are what get color.

## Tuning colors in your theme

Params (`constant.character.escape.rr`) reuse Go's escape scope and inherit the
theme's escape color for free.

The path uses `string.quoted.double.rr`, but note: **many themes (Monokai Pro
included) have a `comment string` rule that repaints any string nested in a
comment back to the comment color** — so a path scoped as a string looks like a
comment until you override it. Force it (and, if you like, the HTTP method) to
the theme's string color with a per-theme `tokenColorCustomizations`:

```jsonc
"editor.tokenColorCustomizations": {
  "[Monokai Pro Light (Filter Sun)]": {
    "textMateRules": [
      { "scope": ["string.quoted.double.rr", "support.constant.http-method.rr"],
        "settings": { "foreground": "#b16803" } }
    ]
  },
  "[Monokai Classic]": {
    "textMateRules": [
      { "scope": ["string.quoted.double.rr", "support.constant.http-method.rr"],
        "settings": { "foreground": "#e6db74" } }
    ]
  }
}
```

Other rr scopes you can retarget the same way: `comment.directive.rr` /
`comment.directive.name.rr` (the muted `api:route` label),
`entity.name.function.reference.rr` (`@refs`), `variable.parameter.config.rr`
(`response=`, `on404=`…).

## Install (from source)

```sh
cd vscode-rr
npx @vscode/vsce package     # produces rr-comments-0.0.1.vsix
code --install-extension rr-comments-0.0.1.vsix
```

Or drop the folder into `~/.vscode/extensions/` and reload.
