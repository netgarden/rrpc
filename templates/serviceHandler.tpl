{{ define "serviceHandler" -}}

class {{ firstLetterToUpper .Name }}ServiceHandler {

    _name: string = "{{ .Name }}";
    _path: string = "{{ .Path }}";

    _client: RRPCClient;

    constructor(client: RRPCClient) {
        this._client = client;
    }

    {{- range $_, $method := .Methods }}
    {{- $pathParams := methodPathParams $method.Path }}

    async {{ firstLetterToLower $method.Name }}(
    {{- range $i, $p := $pathParams }}{{ $p }}: string, {{ end -}}
    {{- range $i, $input := $method.Inputs -}}
    {{ firstLetterToLower $input.Name }}: {{ if $input.IsStruct }}{{ firstLetterToUpper $input.Type }}{{ else }}any{{ end }}
    {{- if not (isLast $i $method.Inputs ) }}, {{ end }}
    {{- end -}}
    ) : Promise<{{ if gt (len $method.Outputs) 0 -}}
    {{- range $i, $output := $method.Outputs -}}
    {{- if $output.IsStruct }}{{ firstLetterToUpper $output.Type }}{{ else }}any{{ end }}
    {{- if not (isLast $i $method.Outputs ) }}, {{ end }}
    {{- end -}}
    {{- else -}}void
    {{- end }}> {

        const method = "{{ $method.Type }}";
        const methodPath = {{ methodPathToTS $method.Path }};
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        {{ if and (eq (len $method.Inputs) 1) ((index $method.Inputs 0).IsStruct) -}}
        const reqDataStr : string = JSON.stringify({{ firstLetterToLower (index $method.Inputs 0).Name }});

        {{ else if gt (len $method.Inputs) 0 -}}
        const reqData = {
            {{- range $i, $input := $method.Inputs }}
            '{{ $input.Name }}': {{ firstLetterToLower $input.Name }},
            {{- end }}
        };

        const reqDataStr : string = JSON.stringify(reqData);

        {{ end -}}
        const res : RRPCResponse = await this._client.sendRequest(method, uri, {{ if gt (len $method.Inputs) 0 }}reqDataStr{{ else }}null{{ end }});

        {{- range $i, $output := $method.Outputs }}
        {{- if $output.IsStruct }}
        const obj : {{ firstLetterToUpper $output.Type }} = await res.response.json();
        {{- else if eq $output.Type "bytes" }}
        const bytes : any = await res.response.text();
        {{- end }}
        {{- end }}

        return
        {{- range $i, $output := $method.Outputs }}
        {{- if $output.IsStruct }} obj
        {{- else if eq $output.Type "bytes" }} bytes
        {{- end -}}
        {{- if not (isLast $i $method.Outputs ) }}, {{ end }}
        {{- end -}}
        ;
    }

    {{- end }}

}

{{- end }}