{{ define "serviceHandler" -}}

class {{ firstLetterToUpper .Name }}ServiceHandler {

    _name: string = "{{ .Name }}";
    _path: string = "{{ .Path }}";

    _client: RRPCClient;

    constructor(client: RRPCClient) {
        this._client = client;
    }

    {{- range $_, $method := .Methods }}

      {{- $getWithStruct := false -}}
      {{- if eq $method.Type "GET" -}}
        {{- if and (eq (len $method.Inputs) 1) ((index $method.Inputs 0).IsStruct) -}}
          {{- $getWithStruct = true -}}
        {{- end -}}
      {{- end }}

      {{- $singleBytes := false -}}
      {{- if and (eq (len $method.Inputs) 1) (eq (index $method.Inputs 0).Type "bytes") -}}
        {{- $singleBytes = true -}}
      {{- end }}

      {{- $isMultipart := eq $method.ContentType "multipart/form-data" -}}

      {{- $pathParams := methodPathParams $method.Path }}

    async {{ firstLetterToLower $method.Name }}(
    {{- range $i, $p := $pathParams }}{{ if $i }}, {{ end }}{{ $p }}: string{{ end -}}
    {{- if and (gt (len $pathParams) 0) (gt (len $method.Inputs) 0) }}, {{ end -}}
    {{- range $i, $input := $method.Inputs }}
    {{- firstLetterToLower $input.Name }}: {{ if $input.IsStruct }}{{ firstLetterToUpper $input.Type }}{{ else }}{{ rrpcDataTypeToTSInputType $input.Type }}{{ end }}
    {{- if not (isLast $i $method.Inputs ) }}, {{ end }}
    {{- end -}}
    ) : Promise<{{ if gt (len $method.Outputs) 0 }}
    {{- range $i, $output := $method.Outputs }}
    {{- if $output.IsStruct }}{{ firstLetterToUpper $output.Type }}{{ else }}{{ rrpcDataTypeToTSType $output.Type }}{{ end }}
    {{- if not (isLast $i $method.Outputs ) }}, {{ end }}
    {{- end }}
    {{- else }}void
    {{- end }}> {

        const method = "{{ $method.Type }}";
        const methodPath = {{ methodPathToTS $method.Path }};
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        let reqUri : string = uri;
        let reqBody : string | Uint8Array | FormData | null = null;

        {{- if $isMultipart }}

        const formData = new FormData();
        {{- range $i, $input := $method.Inputs }}
        {{- if $input.IsStruct }}
        formData.append('{{ $input.Name }}', JSON.stringify({{ firstLetterToLower $input.Name }}));
        {{- end }}
        {{- if eq $input.Type "bytes" }}
        formData.append('{{ $input.Name }}', new Blob([{{ firstLetterToLower $input.Name }} as unknown as BlobPart]));
        {{- end }}
        {{- end }}
        reqBody = formData;

        {{- else if $getWithStruct }}

        const queryParams = new URLSearchParams();
        for (const [key, value] of Object.entries({{ firstLetterToLower (index $method.Inputs 0).Name }} as unknown as Record<string, unknown>)) {
            if (value === null || value === undefined) continue;
            if (value instanceof Uint8Array) {
                queryParams.set(key, btoa(String.fromCharCode(...value)));
            } else {
                queryParams.set(key, String(value));
            }
        }
        reqUri = uri + '?' + queryParams.toString();

        {{- else if $singleBytes }}

        reqBody = {{ firstLetterToLower (index $method.Inputs 0).Name }};

        {{- else if and (eq (len $method.Inputs) 1) ((index $method.Inputs 0).IsStruct) }}

        reqBody = JSON.stringify({{ firstLetterToLower (index $method.Inputs 0).Name }});

        {{- else if gt (len $method.Inputs) 0 }}

        reqBody = JSON.stringify({
            {{- range $i, $input := $method.Inputs }}
            '{{ $input.Name }}': {{ firstLetterToLower $input.Name }},
            {{- end }}
        });

        {{- end }}

        const res : RRPCResponse = await this._client.sendRequest(method, reqUri, reqBody);

        {{ range $i, $output := $method.Outputs -}}
        {{ if $output.IsStruct -}}
        const ret : {{ firstLetterToUpper $output.Type }} = await res.response.json();

        {{ else if eq $output.Type "bytes" -}}
        const ret : Uint8Array = new Uint8Array(await res.response.arrayBuffer());

        {{ else if eq $output.Type "string" -}}
        const ret : string = await res.response.text();

        {{ end -}}
        {{ end -}}

        return
        {{- if gt (len $method.Outputs) 0 }}
        {{- $output := (index $method.Outputs 0) }}
        {{- if or $output.IsStruct (ne $output.Type "") }} ret{{ end -}}
        {{ end -}}
        ;
    }

    {{- end }}

}

{{- end }}