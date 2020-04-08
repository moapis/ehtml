// Copyright (c) 2020, Mohlmann Solutions SRL. All rights reserved.
// Use of this source code is governed by a License that can be found in the LICENSE file.
// SPDX-License-Identifier: BSD-3-Clause

// Package ehtml provides ways of rendering an error html page,
// using Go templates.
package ehtml

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"
)

// Data available in templates.
type Data struct {
	Request    *http.Request // Request object as passed to `Render()`
	StatusCode int
	Message    string
	Data       interface{} // Optional, additional Data
}

// StatusText returns a text for the HTTP status code. It returns the empty
// string if the code is unknown.
func (d *Data) StatusText() string {
	return http.StatusText(d.StatusCode)
}

// String returns the status code, status text and message in a single string.
// For example, in a template:
//   {{ .String }} => 400 Bad Request: Parsing form data
func (d *Data) String() string {
	return fmt.Sprintf("%d %s: %s", d.StatusCode, d.StatusText(), d.Message)
}

// DefaultTmpl is a placeholder template for `Pages.Render()`
const DefaultTmpl = `{{ define "error" -}}
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>{{ .String }}</title>
</head>
<body>
	<h1>{{ .StatusCode }} {{ .StatusText }}</h1>
	<p>{{ .Message }}</p>
</body>
</html>
{{- end -}}
`

var defTmpl = template.Must(template.New("error").Parse(DefaultTmpl))

// Pages allows setting of status page templates.
// Whenever such page needs to be served, a Lookup is done for a template
// named by the code. Eg: "404".
// A generic template named "error" can be provided
// and will be used if there is no status-specific template defined.
//
// If Tmpl is `nil` or no templates are found using above Lookup scheme,
// `DefaultErrTmpl` will be used.
type Pages struct {
	Tmpl *template.Template
}

func (p *Pages) template(status int) *template.Template {
	if p.Tmpl == nil {
		return defTmpl
	}

	if tmpl := p.Tmpl.Lookup(strconv.Itoa(status)); tmpl != nil {
		return tmpl
	}

	if tmpl := p.Tmpl.Lookup("error"); tmpl != nil {
		return tmpl
	}

	return defTmpl
}

var buffers = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// RenderError is returned to the client if the template failed to render.
// This doesn't look nice, but it prevents partial responses.
const RenderError = "500 Internal server error. While handling:\n%s"

// Render a page for passed status code.
// Data and Request are optional and can be nil,
// if the template doesn't need them.
// They are passed to the template as-is.
//
// In case of template execution errors,
// RenderError including the original status and message is sent to the client.
func (p *Pages) Render(w http.ResponseWriter, r *http.Request, statusCode int, msg string, data interface{}) error {
	buf := buffers.Get().(*bytes.Buffer)
	defer buffers.Put(buf)

	d := &Data{
		Request:    r,
		StatusCode: statusCode,
		Message:    msg,
		Data:       data,
	}

	if err := p.template(statusCode).Execute(buf, d); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, RenderError, d)

		return fmt.Errorf("ehtml Render template: %w", err)
	}

	w.WriteHeader(statusCode)
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("ehtml Render, write to client: %w", err)
	}
	return nil
}
