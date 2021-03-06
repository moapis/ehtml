[![Build Status](https://travis-ci.org/moapis/ehtml.svg?branch=master)](https://travis-ci.org/moapis/ehtml)
[![codecov](https://codecov.io/gh/moapis/ehtml/branch/master/graph/badge.svg)](https://codecov.io/gh/moapis/ehtml)
[![Go Report Card](https://goreportcard.com/badge/github.com/moapis/ehtml)](https://goreportcard.com/report/github.com/moapis/ehtml)
[![GoDoc](https://godoc.org/github.com/moapis/ehtml?status.svg)](https://godoc.org/github.com/moapis/ehtml)

# eHTML

Package ehtml provides ways of rendering an error html page, using Go templates.

Too many time we've found ourselves writing customized error pages. With this package we hope to relieve ourselves from this trivial, but yet repetetive job.

Simply load your templates into the object and call `Render()` with the request, status code and optional data. 

Writing a generic web server? Waiting for templates from your designer? No problem! A default placeholder template will be used instead.

## Example

Define some templates:

````
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
````

Parse them into a globale variable (or part of your Handler object). One can also use `ParseFiles()` or `ParseGlob()`:

````
var errorPages = &Pages{template.Must(template.New("error").Parse(templates))}
````

If you are using Gorilla mux, set the `NotFoundHandler`

````
rtr := mux.NewRouter()
rtr.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    if err := p.Render(w, &Data{Req: r, Code: http.StatusNotFound}); err != nil {
        log.Println(err)
    }
})
````

Optionally, extend `Data` to add more context.
As an alternative, you can also roll your own implementation of `Provider`.

````
type data struct {
    Data
    ReqID int
}
````

And whenever something goes wrong in your handlers, call `Render()`:

````
err := p.Render(w, &data{Data{req, http.StatusInternalServerError, "DB connection"}, 666})
if err != nil {
    log.Println(err)
}
````

## License

BSD 3 Clause.
