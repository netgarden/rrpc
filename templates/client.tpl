{{ define "client" -}}

class Client {

    _client: RRPCClient;

    {{ range $_, $service := .Services -}}
    {{ firstLetterToLower $service.Name }}: {{ firstLetterToUpper $service.Name }}ServiceHandler;
    {{ end }}

    constructor(url: string) {
        this._client = new RRPCClient(url);

        {{ range $_, $service := .Services -}}
        this.{{ firstLetterToLower $service.Name }} = new {{ firstLetterToUpper $service.Name }}ServiceHandler(this._client);
        {{ end }}
    }

}

{{- end }}