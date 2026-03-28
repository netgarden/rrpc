{{ template "client" . }}

{{ range $_, $service := .Services }}
{{ template "serviceHandler" $service }}
{{ end }}

{{ range $_, $struct := .Structs }}
{{ template "struct" $struct }}
{{ end }}

{{ template "rrpcClient" . }}