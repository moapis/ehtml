// Copyright (c) 2020, Mohlmann Solutions SRL. All rights reserved.
// Use of this source code is governed by a License that can be found in the LICENSE file.
// SPDX-License-Identifier: BSD-3-Clause

package ehtml

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestData_StatusText(t *testing.T) {
	tests := []struct {
		name       string
		StatusCode int
		want       string
	}{
		{
			"Unknown",
			900,
			"",
		},
		{
			"Known",
			http.StatusTeapot,
			"I'm a teapot",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Data{
				StatusCode: tt.StatusCode,
			}
			if got := d.StatusText(); got != tt.want {
				t.Errorf("Data.StatusText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestData_String(t *testing.T) {
	type fields struct {
		StatusCode int
		Message    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"Code not set",
			fields{
				Message: "Something's missing",
			},
			"0 : Something's missing",
		},
		{
			"Known",
			fields{
				StatusCode: http.StatusBadRequest,
				Message:    "Parsing form data",
			},
			"400 Bad Request: Parsing form data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Data{
				StatusCode: tt.fields.StatusCode,
				Message:    tt.fields.Message,
			}
			if got := d.String(); got != tt.want {
				t.Errorf("Data.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestData_Title(t *testing.T) {
	d := &Data{
		StatusCode: http.StatusBadRequest,
		Message:    "Parsing form data",
	}
	want := "400 Bad Request: Parsing form data"

	if got := d.Title(); got != want {
		t.Errorf("Data.Title() = %v, want %v", got, want)
	}
}

const defaultTmplOut = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>404 Not Found: Foo bar</title>
</head>
<body>
	<h1>404 Not Found</h1>
	<p>Foo bar</p>
</body>
</html>`

const (
	testErrTemplate   = `{{ define "error" }}Generic template{{ end }}`
	test404Template   = `{{ define "404" }}404 template{{ end }}`
	testWrongTemplate = `{{ define "wrong" }}Wrong template{{ end }}`
)

var testTmpl, wrongTmpl *template.Template

func init() {
	testTmpl = template.Must(template.New("error").Parse(testErrTemplate))
	testTmpl = template.Must(testTmpl.Parse(test404Template))

	wrongTmpl = template.Must(template.New("wrong").Parse(testWrongTemplate))
}

func TestPages_template(t *testing.T) {
	data := &Data{
		StatusCode: 404,
		Message:    "Foo bar",
	}

	tests := []struct {
		name   string
		tmpl   *template.Template
		status int
		want   string
	}{
		{
			"Nil, default",
			nil,
			404,
			defaultTmplOut,
		},
		{
			"Code defined",
			testTmpl,
			404,
			"404 template",
		},
		{
			"Unknown code, generic",
			testTmpl,
			400,
			"Generic template",
		},
		{
			"Wrong, default",
			wrongTmpl,
			404,
			defaultTmplOut,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pages{
				Tmpl: tt.tmpl,
			}

			var buf bytes.Buffer

			if err := p.template(tt.status).Execute(&buf, data); err != nil {
				t.Fatal(err)
			}

			if got := buf.String(); got != tt.want {
				t.Errorf("Pages.template() = \n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}

func TestPages_Render(t *testing.T) {
	errTmpl := template.Must(template.New("error").Parse("{{ .Missing }}"))

	tests := []struct {
		name       string
		tmpl       *template.Template
		statusCode int
		want       string
		wantCode   int
		wantErr    bool
	}{
		{
			"Default template",
			nil,
			http.StatusNotFound,
			defaultTmplOut,
			http.StatusNotFound,
			false,
		},
		{
			"Execution error",
			errTmpl,
			http.StatusNotFound,
			"500 Internal server error. While handling:\n404 Not Found: Foo bar",
			http.StatusInternalServerError,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pages{
				Tmpl: tt.tmpl,
			}

			req := httptest.NewRequest("GET", "http://example.com/foo", nil)
			w := httptest.NewRecorder()

			if err := p.Render(w, req, tt.statusCode, "Foo bar", nil); (err != nil) != tt.wantErr {
				t.Fatalf("Pages.Render() error = %v, wantErr %v", err, tt.wantErr)
			}

			resp := w.Result()
			body, _ := ioutil.ReadAll(resp.Body)

			if resp.StatusCode != tt.wantCode {
				t.Errorf("Pages.Render() status = %v, want: %v", resp.StatusCode, tt.wantCode)
			}

			got := string(body)
			if got != tt.want {
				t.Errorf("Pages.Render() = \n%v\nwant\n%v", got, tt.want)
			}
		})
	}
}

type errorWriter struct{}

func (errorWriter) Header() http.Header       { return nil }
func (errorWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
func (errorWriter) WriteHeader(int)           {}

func TestPages_Render_WriteError(t *testing.T) {
	p := &Pages{}
	if err := p.Render(errorWriter{}, nil, 404, "Foo bar", nil); !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("Pages.Render() error = %v, wantErr %v", err, io.ErrClosedPipe)
	}
}

const exampleTemplates = `
{{- define "head" -}}
<head>
	<meta charset="utf-8">
	<title>{{ .String }}</title>
</head>
{{- end -}}

{{- define "error" -}}
<!DOCTYPE html>
<html lang="en">
{{ template "head" }}
<body>
	<h1>{{ .StatusCode }} {{ .StatusText }}</h1>
	<p>
		{{ .Message }} while serving {{ .Request.URL.Path }}.
		Request ID: {{ .Data.RequestID }}
	</p>
	<p><i>This is a generic error page</i><p>
</body>
</html>
{{- end -}}

{{- define "500" -}}
<!DOCTYPE html>
<html lang="en">
{{ template "head" }}
<body>
	<h1>Snap!</h1>
	<h2>{{ .StatusCode }} {{ .StatusText }}</h2>
	<p>
		Something went really wrong and we've been notified!
		Please try again later.
	</p>
	<p><i>
		Error: {{ .Message }} while serving {{ .Request.URL.Path }}.
		Request ID: {{ .Data.RequestID }}
	</i></p>
</body>
</html>
{{- end -}}

{{- define "404" -}}
<!DOCTYPE html>
<html lang="en">
<head>
	{{ template "head" }}
</head>
<body>
	<h1>{{ .StatusCode }} {{ .StatusText }}</h1>
	<p>
		{{ .Request.URL.Path }} could not be found.
</body>
</html>
{{- end -}}`

func Example() {
	p := &Pages{template.Must(template.New("error").Parse(exampleTemplates))}

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Serves the client with the "500" template
	err := p.Render(w, req, http.StatusInternalServerError, "DB connection", struct{ RequestID int }{666})
	if err != nil {
		log.Println(err)
	}

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))

	w = httptest.NewRecorder()

	// 400 is not defined, so the generic "error" template is used instead.
	err = p.Render(w, req, http.StatusBadRequest, "Missing token in URL", struct{ RequestID int }{667})
	if err != nil {
		log.Println(err)
	}

	resp = w.Result()
	body, _ = ioutil.ReadAll(resp.Body)

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
}

func Example_notFoundHandler() {
	p := &Pages{template.Must(template.New("error").Parse(exampleTemplates))}

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := p.Render(w, r, http.StatusNotFound, "", nil); err != nil {
			log.Println(err)
		}
	})
}
