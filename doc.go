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
	{{- end -}}

Parse them into a globale variable (or part of your Handler object):

	var errorPages = &Pages{template.Must(template.New("error").Parse(templates))}

If you are using Gorilla mux, set the `NotFoundHandler`

	rtr := mux.NewRouter()
	rtr.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := p.Render(w, &Data{Req: r, Code: http.StatusNotFound}); err != nil {
			log.Println(err)
		}
	})

Optionally, extend `Data` to add more context.
As an alternative, you can also roll your own implementation of `Provider`.

	type data struct {
		Data
		ReqID int
	}

And whenever something goes wrong in your handlers, call `Render()`:

	err := p.Render(w, &data{Data{req, http.StatusInternalServerError, "DB connection"}, 666})
	if err != nil {
		log.Println(err)
	}
*/
package ehtml
