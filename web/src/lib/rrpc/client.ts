export class RRPCError {

    public error: string;
    public code: number;
    public message: string;
    public cause: string | null;

    constructor(error : string, code : number, message : string, cause : string | null) {
        this.error = error;
        this.code = code;
        this.message = message;
        this.cause = cause;
    }

}

interface ErrorData {
    error : string
    code : number
    msg : string
    cause? : string
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
                const errObj : ErrorData = await resp.json();
                this.handleError(errObj);
            }
            throw new UnexpectedReturnCodeError(resp.status, resp.statusText);
        }

        return new RRPCResponse(resp);
    }

    private handleError(error : ErrorData) : void {
        console.log("Error:", error);
        throw new RRPCError(
            error.error,
            error.code,
            error.msg,
            error.cause ? error.cause : null
        );
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