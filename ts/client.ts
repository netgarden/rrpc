// escapePathValue encodes value for safe inclusion in a URL path built from
// a "path"-typed parameter, which — unlike a "string"-typed one — may
// itself contain "/" separators that must be preserved rather than
// escaped away. Each segment between slashes is percent-encoded
// individually; the separators themselves are not.
export function escapePathValue(value: string): string {
    return value.split('/').map(encodeURIComponent).join('/');
}

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
    _accessToken: string | null = null;

    constructor(url: string) {
        this._url = url;
    }

    // setAccessToken attaches an "Authorization: Bearer <token>" header to
    // every subsequent request. Pass null to stop sending it (e.g. on logout).
    setAccessToken(token: string | null): void {
        this._accessToken = token;
    }

    async sendRequest(method: string, uri: string, data: string | Uint8Array | FormData | null): Promise<RRPCResponse> {

        const headers : Record<string, string> = {
            'Accept': 'application/json',
        }

        if (this._accessToken) {
            headers['Authorization'] = 'Bearer ' + this._accessToken;
        }

        let body : BodyInit | null = null;
        if (data instanceof FormData) {
            // do not set Content-Type; browser sets it with the correct boundary
            body = data;
        } else if (data !== null && typeof data !== 'string') {
            headers['Content-Type'] = 'application/octet-stream';
            body = data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength) as ArrayBuffer;
        } else if (data != null && data !== "") {
            headers['Content-Type'] = 'application/json';
            body = data;
        }

        const base : string = this._url.endsWith('/') ? this._url.slice(0, -1) : this._url;
        const path : string = uri.startsWith('/') ? uri : '/' + uri;
        const resp : Response = await fetch(
            base + path,
            {
                headers: headers,
                method: method,
                body: body,
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