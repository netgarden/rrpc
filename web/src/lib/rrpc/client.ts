export class RRPCClient {
    _url: string;

    constructor(url: string) {
        this._url = url;
    }

    async sendRequest(method: string, uri: string, data: string | null): Promise<RRPCResponse> {

        const headers : Record<string, string> = {
            'Accept': 'application/json',
        }

        if (data != null && data !== "") {
            headers['Content-Type'] = 'application/json';
        }

        const resp : Response = await fetch(
            this._url + "/" + uri,
            {
                headers: headers,
                method: method,
                body: data,
            }
        );

        if (resp.status !== 200) {

        }

        return new RRPCResponse(resp);
    }

}

export class RRPCResponse {

    public response: Response;

    constructor(response: Response) {
        this.response = response;
    }

}