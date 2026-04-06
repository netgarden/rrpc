export interface RRPCError {
    error: string;
    code: number;
    msg: string;
    cause?: string;
    status: number;
}

export class RRPCClient {
    _url: string;

    constructor(url: string) {
        this._url = url;
    }

    async sendRequest(method: string, uri: string, data: string | Uint8Array | null): Promise<RRPCResponse> {

        const headers : Record<string, string> = {
            'Accept': 'application/json',
        }

        if (data instanceof Uint8Array) {
            headers['Content-Type'] = 'application/octet-stream';
        } else if (data != null && data !== "") {
            headers['Content-Type'] = 'application/json';
        }

        const base : string = this._url.endsWith('/') ? this._url.slice(0, -1) : this._url;
        const path : string = uri.startsWith('/') ? uri : '/' + uri;
        const resp : Response = await fetch(
            base + path,
            {
                headers: headers,
                method: method,
                body: data,
            }
        );

        if (resp.status >= 400) {
            const contentType : string = resp.headers.get('Content-Type') ?? '';
            if (contentType.includes('application/problem+json')) {
                const errBody = await resp.json();
                throw errBody;
            }
            throw new UnexpectedReturnCodeError(resp.status, resp.statusText);
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

export class UnexpectedReturnCodeError extends Error {

    code : number;
    reason : string;

    constructor(code : number, reason : string) {
        super("Received unexpected return code: '" + code + "' with reason '" + reason + "'")
        this.code = code;
        this.reason = reason;
    }

}