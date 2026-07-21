# rr directive highlighting

Syntax highlighting for rr `//api:` comment directives in Go source.

It's an **injection** grammar — it colors tokens *inside* Go comments, so
normal Go highlighting is untouched. Nothing else in your `.go` files changes.

## What it colors

Line directives:

```go
//api:central response=json onerror=@handleError on404=notFound on400=@badRequest
//api:route GET /api/posts/{slug=@isSlug} -- func checker takes what Atoi rejected
//api:route GET /api/files/{path...}
//api:middleware @requireToken
//api:errorhandler @postNotFound
//api:response json
```

Inline directives:

```go
func (u *UsersApi) Create( /** api:body json */ u User) {}
func (s *SearchApi) Search( /* api:query @parseSearch */ q SearchQuery, /* api:header X-Request-Id */ rid string) {}
```

Token scopes:

| token | scope |
| --- | --- |
| `api:` prefix | `comment.directive.rr` |
| directive name (`central`, `route`, `body`, …) | `comment.directive.name.rr` |
| HTTP method | `support.constant.http-method.rr` |
| `key=` (response, onerror, on400…) | `variable.parameter.config.rr` |
| `@ref` | `entity.name.function.reference.rr` |
| `{name}` / `...` inside a path param | `constant.character.escape.rr` (same as `\d` in a Go string) |
| `/path/segments` | `string.quoted.double.rr` (same as a Go string) |
| `*` catch-all | `keyword.operator.wildcard.rr` |
| `-- comment` suffix | `comment.line.double-dash.rr` |

The `api:` prefix and directive name deliberately sit in the `comment.*` scope
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
