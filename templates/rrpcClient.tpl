{{ define "rrpcClient" -}}

class RRPCClient {
    _url: string;

    constructor(url: string) {
        this._url = url;
    }

    async sendRequest(method: string, uri: string, data: string): string {

        let headers = {
            'Accept': 'application/json',
        }

        if (data != null && data !== "") {
            headers['Content-Type'] = 'application/json';
        }

        let resp = await fetch(
            this._url + "/" + uri,
            {
                headers: headers,
                method: method,
                body: data,
            }
        );

        if (resp.status() !== 200) {
            
        }

        // TODO: ...

    }

}

{{- end }}