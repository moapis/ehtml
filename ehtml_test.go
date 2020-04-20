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
	"reflect"
	"testing"

	"github.com/gorilla/mux"
)

func TestStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   string
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
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Data.StatusText() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatus_int(t *testing.T) {
	tests := []struct {
		name string
		s    Status
		want int
	}{
		{
			"Int",
			400,
			400,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.Int(); got != tt.want {
				t.Errorf("Status.int() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStatus_toA(t *testing.T) {
	tests := []struct {
		name string
		s    Status
		want string
	}{
		{
			"String",
			400,
			"400",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.toA(); got != tt.want {
				t.Errorf("Status.toA() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestData_Request(t *testing.T) {
	d := &Data{Req: httptest.NewRequest("GET", "http://example.com/foo", nil)}
	if got := d.Request(); !reflect.DeepEqual(got, d.Req) {
		t.Errorf("Data.Request() = %v, want %v", got, d.Req)
	}

}

func TestData_Status(t *testing.T) {
	d := &Data{Code: http.StatusTeapot}
	if got := d.Status(); got != http.StatusTeapot {
		t.Errorf("Data.Status() = %v, want %v", got, http.StatusTeapot)
	}
}

func TestData_Message(t *testing.T) {
	d := &Data{Msg: "FooBar"}
	if got := d.Message(); got != "FooBar" {
		t.Errorf("Data.Message() = %v, want %v", got, "FooBar")
	}
}

func TestData_String(t *testing.T) {
	type fields struct {
		Code Status
		Msg  string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			"Code not set",
			fields{
				Msg: "Something's missing",
			},
			"0 : Something's missing",
		},
		{
			"Known",
			fields{
				Code: http.StatusBadRequest,
				Msg:  "Parsing form data",
			},
			"400 Bad Request: Parsing form data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Data{
				Code: tt.fields.Code,
				Msg:  tt.fields.Msg,
			}
			if got := d.String(); got != tt.want {
				t.Errorf("Data.String() = %v, want %v", got, tt.want)
			}
		})
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
	d := &Data{
		Code: 404,
		Msg:  "Foo bar",
	}

	tests := []struct {
		name   string
		tmpl   *template.Template
		status Status
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

			if err := p.template(tt.status).Execute(&buf, d); err != nil {
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
		name     string
		tmpl     *template.Template
		code     Status
		want     string
		wantCode int
		wantErr  bool
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

			d := &Data{
				Req:  httptest.NewRequest("GET", "http://example.com/foo", nil),
				Code: tt.code,
				Msg:  "Foo bar",
			}

			w := httptest.NewRecorder()

			if err := p.Render(w, d); (err != nil) != tt.wantErr {
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
	d := &Data{
		Req:  httptest.NewRequest("GET", "http://example.com/foo", nil),
		Code: http.StatusTeapot,
		Msg:  "Foo bar",
	}
	if err := p.Render(errorWriter{}, d); !errors.Is(err, io.ErrClosedPipe) {
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
{{ template "head" . }}
<body>
	<h1>{{ .Status.Int }} {{ .Status }}</h1>
	<p>
		{{ .Message }} while serving {{ .Request.URL.Path }}.
		Request ID: {{ .ReqID }}
	</p>
	<p><i>This is a generic error page</i><p>
</body>
</html>
{{- end -}}

{{- define "500" -}}
<!DOCTYPE html>
<html lang="en">
{{ template "head" . }}
<body>
	<h1>Snap!</h1>
	<h2>{{ .Status.Int }} {{ .Status }}</h2>
	<p>
		Something went really wrong and we've been notified!
		Please try again later.
	</p>
	<p><i>
		Error: {{ .Message }} while serving {{ .Request.URL.Path }}.
		Request ID: {{ .ReqID }}
	</i></p>
</body>
</html>
{{- end -}}

{{- define "404" -}}
<!DOCTYPE html>
<html lang="en">
<head>
	{{ template "head" . }}
</head>
<body>
	<h1>{{ .Status.Int}} {{ .Status }}</h1>
	<p>
		{{ .Request.URL.Path }} could not be found.
	</p>
</body>
</html>
{{- end -}}`

func Example() {
	p := &Pages{template.Must(template.New("error").Parse(exampleTemplates))}

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Extend Data, as needed
	type data struct {
		Data
		ReqID int
	}

	// Serves the client with the "500" template
	err := p.Render(w, &data{Data{req, http.StatusInternalServerError, "DB connection"}, 666})
	if err != nil {
		log.Println(err)
	}

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))

	w = httptest.NewRecorder()

	// 400 is not defined, so the generic "error" template is used instead.
	err = p.Render(w, &data{Data{req, http.StatusBadRequest, "Missing token in URL"}, 667})
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

	rtr := mux.NewRouter()
	rtr.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := p.Render(w, &Data{Req: r, Code: http.StatusNotFound}); err != nil {
			log.Println(err)
		}
	})
}
