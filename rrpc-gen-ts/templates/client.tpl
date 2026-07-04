{{ define "client" -}}

export class {{ firstLetterToUpper .Name }}Client {

    _client: RRPCClient;

    {{ range $_, $service := .Services -}}
    {{ firstLetterToLower $service.Name }}Service: {{ firstLetterToUpper $service.Name }}ServiceHandler;
    {{ end }}

    constructor(url: string) {
        this._client = new RRPCClient(url);

        {{ range $_, $service := .Services -}}
        this.{{ firstLetterToLower $service.Name }}Service = new {{ firstLetterToUpper $service.Name }}ServiceHandler(this._client);
        {{ end }}
    }

}

{{- end }}