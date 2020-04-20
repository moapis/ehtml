// Copyright (c) 2020, Mohlmann Solutions SRL. All rights reserved.
// Use of this source code is governed by a License that can be found in the LICENSE file.
// SPDX-License-Identifier: BSD-3-Clause

package ehtml

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"sync"
)

// Status holds an HTTP status code
type Status int

// String returns the text descriptiom for the HTTP status code.
// It returns the empty string if the code is unknown.
func (s Status) String() string { return http.StatusText(int(s)) }

// Int returns Status as int
func (s Status) Int() int { return int(s) }

func (s Status) toA() string { return strconv.Itoa(s.Int()) }

// Provider of data to templates
type Provider interface {
	// Request returns the incomming http Request object
	Request() *http.Request
	Status() Status
	Message() string
	// String returns the status code, status text and message in a single string.
	// For example: "400 Bad Request: Parsing form data"
	String() string
}

// Data can be used as a default or embedded type to implement Provider.
type Data struct {
	Req  *http.Request
	Code Status
	Msg  string
}

// Request implements Provider
func (d *Data) Request() *http.Request { return d.Req }

// Status implements Provider
func (d *Data) Status() Status { return d.Code }

// Message implements Provider
func (d *Data) Message() string { return d.Msg }

func (d *Data) String() string {
	return fmt.Sprintf("%d %s: %s", d.Code, d.Code, d.Msg)
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
	<h1>{{ .Status.Int }} {{ .Status }}</h1>
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

func (p *Pages) template(s Status) *template.Template {
	if p.Tmpl == nil {
		return defTmpl
	}

	if tmpl := p.Tmpl.Lookup(s.toA()); tmpl != nil {
		return tmpl
	}

	if tmpl := p.Tmpl.Lookup("error"); tmpl != nil {
		return tmpl
	}

	return defTmpl
}

type bufPool struct {
	p sync.Pool
}

func (p *bufPool) Get() *bytes.Buffer {
	if b, ok := p.p.Get().(*bytes.Buffer); ok {
		return b
	}

	return new(bytes.Buffer)
}

func (p *bufPool) Put(b *bytes.Buffer) {
	b.Reset()
	p.p.Put(b)
}

var buffers = &bufPool{}

// RenderError is returned to the client if the template failed to render.
// This doesn't look nice, but it prevents partial responses.
const RenderError = "500 Internal server error. While handling:\n%s"

// Render a page for passed status code.
// In case of template execution errors,
// "RenderError" including the original status and message is sent to the client.
func (p *Pages) Render(w http.ResponseWriter, dp Provider) error {
	buf := buffers.Get()
	defer buffers.Put(buf)

	if err := p.template(dp.Status()).Execute(buf, dp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, RenderError, dp)

		return fmt.Errorf("ehtml Render template: %w", err)
	}

	w.WriteHeader(dp.Status().Int())
	if _, err := buf.WriteTo(w); err != nil {
		return fmt.Errorf("ehtml Render, write to client: %w", err)
	}
	return nil
}
