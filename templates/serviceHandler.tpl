{{ define "serviceHandler" -}}

class {{ firstLetterToUpper .Name }}ServiceHandler {

    _name: string = "{{ .Name }}";
    _path: string = "{{ .Path }}";

    _client: RRPCClient;

    constructor(client: RRPCClient) {
        this._client = client;
    }

    {{ range $_, $method := .Methods -}}

    {{ firstLetterToLower $method.Name }}(
    {{- range $i, $input := $method.Inputs -}}
    {{ firstLetterToLower $input.Name }}: {{ if $input.IsStruct }}{{ firstLetterToUpper $input.Type }}{{ else }}any{{ end }}
    {{- if not (isLast $i $method.Inputs ) }}, {{ end }}
    {{- end -}}
    )
    {{- if gt (len $method.Outputs) 0 -}}
    : {{ range $i, $output := $method.Outputs -}}
    {{- if $output.IsStruct }}{{ firstLetterToUpper $output.Type }}{{ else }}any{{ end }}
    {{- if not (isLast $i $method.Outputs ) }},{{ end }}
    {{- end -}}
    {{- end }} {

        let method = "{{ $method.Type }}";
        let path = "{{ $method.Path }}"
        let uri = "/" + this._path + "/" + path;

        {{ if gt (len $method.Inputs) 0 }}
        let reqData = {
            {{- range $i, $input := $method.Inputs }}
            '{{ $input.Name }}': {{ firstLetterToLower $input.Name }},
            {{- end }}
        };

        let reqDataStr = JSON.stringify(reqData);
        {{ end }}

        // TODO: serialization, deserialization, ...
        let resp = this._client.sendRequest(method, uri, {{ if gt (len $method.Inputs) 0 }}reqDataStr{{ else }}null{{ end }});

        {{ if gt (len $method.Outputs) 0 }}
        {{ end }}

    }

    {{ end }}

}

{{- end }}