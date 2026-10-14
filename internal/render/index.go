// Package render generates HTML pages.
package render

import (
	"html/template"
	"io"

	"github.com/onozaty/dbfolio/internal/input/tbls"
)

// Minimal placeholder page for Phase 1; replaced by embedded templates in Phase 2.
var indexTmpl = template.Must(template.New("index").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{with .Name}}{{.}} - {{end}}dbfolio</title>
</head>
<body>
<h1>{{with .Name}}{{.}}{{else}}Database{{end}}</h1>
{{with .Desc}}<p>{{.}}</p>{{end}}
<h2>Tables ({{len .Tables}})</h2>
<table>
<tr><th>Name</th><th>Type</th><th>Columns</th><th>Comment</th></tr>
{{range .Tables}}<tr><td>{{.Name}}</td><td>{{.Type}}</td><td>{{len .Columns}}</td><td>{{.Comment}}</td></tr>
{{end}}</table>
</body>
</html>
`))

// Index writes the overview page.
func Index(w io.Writer, s *tbls.Schema) error {
	return indexTmpl.Execute(w, s)
}
