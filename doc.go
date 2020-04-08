// Copyright (c) 2020, Mohlmann Solutions SRL. All rights reserved.
// Use of this source code is governed by a License that can be found in the LICENSE file.
// SPDX-License-Identifier: BSD-3-Clause

/*
Package ehtml provides ways of rendering an error html page, using Go templates.
It supports status code specific templates, with fallback to a generic error template.
If no templates are defined, it uses a simple placeholder template.

Define some templates:

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
	{{- end -}}

Parse them into a globale variable (or part of your Handler object):

	var errorPages = &Pages{template.Must(template.New("error").Parse(templates))}

If you are using Gorilla mux, set the `NotFoundHandler`

	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := p.Render(w, r, http.StatusNotFound, "", nil); err != nil {
			log.Println(err)
		}
	})

And whenever something goes wrong in your handlers, call `Render()`:

	err := p.Render(w, req, http.StatusInternalServerError, "DB connection", struct{ RequestID int }{666})
	if err != nil {
		log.Println(err)
	}
*/
package ehtml
