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

    // contentType, when given, is used as-is instead of the guess below - set
    // by generated code from the method's declared Content-Type (auto-inferred,
    // or overridden by @RequestContentType; see rrpc's README). Left
    // undefined/empty, behavior is unchanged. Never applies to a FormData
    // body: the browser must set that Content-Type itself, since it carries
    // the multipart boundary (a multipart bytes field's own part-level
    // Content-Type, from @RequestContentType on a multipart method, is set
    // on that part's Blob instead - see the generated client call site).
    async sendRequest(method: string, uri: string, data: string | Uint8Array | FormData | null, contentType?: string): Promise<RRPCResponse> {

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
            headers['Content-Type'] = contentType || 'application/octet-stream';
            body = data.buffer.slice(data.byteOffset, data.byteOffset + data.byteLength) as ArrayBuffer;
        } else if (data != null && data !== "") {
            headers['Content-Type'] = contentType || 'application/json';
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

    // openStream/openInStream/openOutStream dial a "@Stream"/
    // "@InputStream"/"@OutputStream" method as a WebSocket. The token
    // (if set via setAccessToken) rides as a "bearer, <token>"
    // subprotocol rather than an Authorization header, since the native
    // WebSocket constructor can't set arbitrary headers on the handshake
    // request - see maf/rrpc-auth's matching Sec-WebSocket-Protocol
    // fallback.
    //
    // raw selects wire format, not just the TS type: false (the default)
    // means typed JSON-per-frame messages (with a {"data"|"error": ...}
    // envelope on the server->client direction, matching a declared
    // struct message type); true means raw binary frames with no
    // envelope at all, for a direction declared "bytes". Generated code
    // passes it based on what was declared in the .rrpc source - callers
    // should not need to set it directly.

    openStream<In, Out>(uri: string, raw: boolean = false): Promise<RRPCStream<In, Out>> {
        return new Promise((resolve, reject) => {
            const socket = this.dialStreamSocket(uri, raw);
            socket.addEventListener('open', () => resolve(new RRPCStream<In, Out>(socket, raw)), { once: true });
            socket.addEventListener('error', () => reject(new Error('WebSocket connection failed')), { once: true });
        });
    }

    openInStream<In>(uri: string, raw: boolean = false): Promise<RRPCInStream<In>> {
        return new Promise((resolve, reject) => {
            const socket = this.dialStreamSocket(uri, raw);
            socket.addEventListener('open', () => resolve(new RRPCInStream<In>(socket, raw)), { once: true });
            socket.addEventListener('error', () => reject(new Error('WebSocket connection failed')), { once: true });
        });
    }

    openOutStream<Out>(uri: string, raw: boolean = false): Promise<RRPCOutStream<Out>> {
        return new Promise((resolve, reject) => {
            const socket = this.dialStreamSocket(uri, raw);
            socket.addEventListener('open', () => resolve(new RRPCOutStream<Out>(socket, raw)), { once: true });
            socket.addEventListener('error', () => reject(new Error('WebSocket connection failed')), { once: true });
        });
    }

    private dialStreamSocket(uri: string, raw: boolean): WebSocket {
        const wsBase : string = this._url.replace(/^http/, 'ws');
        const base : string = wsBase.endsWith('/') ? wsBase.slice(0, -1) : wsBase;
        const path : string = uri.startsWith('/') ? uri : '/' + uri;
        const protocols : string[] | undefined = this._accessToken ? ['bearer', this._accessToken] : undefined;
        const socket = new WebSocket(base + path, protocols);
        if (raw) {
            socket.binaryType = 'arraybuffer';
        }
        return socket;
    }

}

interface StreamEnvelope<T> {
    data?: T;
    error?: ErrorData;
}

// _WSMessageQueue is the shared incoming-message buffering/async-iterator
// implementation behind RRPCStream/RRPCOutStream's messages() - not
// exported, used internally by both.
class _WSMessageQueue<T> {

    private raw: boolean;
    private buffer: (T | RRPCError)[] = [];
    private waiting: (() => void)[] = [];
    private ended = false;
    private endError: Error | null = null;

    constructor(socket: WebSocket, raw: boolean) {
        this.raw = raw;
        socket.addEventListener('message', (ev: MessageEvent) => {
            this.buffer.push(this.decode(ev.data));
            this.wake();
        });
        socket.addEventListener('close', (ev: CloseEvent) => {
            this.ended = true;
            if (!ev.wasClean) {
                this.endError = new Error(`WebSocket closed abnormally (code ${ev.code}: ${ev.reason})`);
            }
            this.wake();
        });
        socket.addEventListener('error', () => {
            this.ended = true;
            if (!this.endError) {
                this.endError = new Error('WebSocket error');
            }
            this.wake();
        });
    }

    private decode(data: unknown): T | RRPCError {
        if (this.raw) {
            return new Uint8Array(data as ArrayBuffer) as unknown as T;
        }
        const envelope = JSON.parse(data as string) as StreamEnvelope<T>;
        if (envelope.error) {
            return new RRPCError(envelope.error.error, envelope.error.code, envelope.error.msg, envelope.error.cause ?? null);
        }
        return envelope.data as T;
    }

    private wake(): void {
        const w = this.waiting.shift();
        if (w) w();
    }

    private wait(): Promise<void> {
        return new Promise((resolve) => { this.waiting.push(resolve); });
    }

    async *messages(): AsyncGenerator<T> {
        while (true) {
            while (this.buffer.length === 0 && !this.ended) {
                await this.wait();
            }
            if (this.buffer.length > 0) {
                const item = this.buffer.shift() as T | RRPCError;
                if (item instanceof RRPCError) {
                    throw item;
                }
                yield item;
                continue;
            }
            if (this.endError) {
                throw this.endError;
            }
            return;
        }
    }
}

// RRPCStream is the client side of a bidirectional "@Stream" method.
export class RRPCStream<In, Out> {

    private socket: WebSocket;
    private raw: boolean;
    private queue: _WSMessageQueue<Out>;

    constructor(socket: WebSocket, raw: boolean) {
        this.socket = socket;
        this.raw = raw;
        this.queue = new _WSMessageQueue<Out>(socket, raw);
    }

    send(msg: In): void {
        if (this.raw) {
            this.socket.send(msg as unknown as Uint8Array<ArrayBuffer>);
        } else {
            this.socket.send(JSON.stringify(msg));
        }
    }

    // messages() yields each server->client message in order; it throws
    // an RRPCError if the server sends a typed error frame (see
    // rrpc.Stream.SendError on the Go side), throws a plain Error on an
    // abnormal close, and returns (ending the loop) on a normal one.
    messages(): AsyncGenerator<Out> {
        return this.queue.messages();
    }

    close(code?: number, reason?: string): void {
        this.socket.close(code, reason);
    }
}

// RRPCInStream is the client side of a client->server-only "@InputStream"
// method - the client only sends, so there's no messages().
export class RRPCInStream<In> {

    private socket: WebSocket;
    private raw: boolean;

    constructor(socket: WebSocket, raw: boolean) {
        this.socket = socket;
        this.raw = raw;
    }

    send(msg: In): void {
        if (this.raw) {
            this.socket.send(msg as unknown as Uint8Array<ArrayBuffer>);
        } else {
            this.socket.send(JSON.stringify(msg));
        }
    }

    close(code?: number, reason?: string): void {
        this.socket.close(code, reason);
    }
}

// RRPCOutStream is the client side of a server->client-only
// "@OutputStream" method - the client only receives, so there's no send().
export class RRPCOutStream<Out> {

    private socket: WebSocket;
    private queue: _WSMessageQueue<Out>;

    constructor(socket: WebSocket, raw: boolean) {
        this.socket = socket;
        this.queue = new _WSMessageQueue<Out>(socket, raw);
    }

    messages(): AsyncGenerator<Out> {
        return this.queue.messages();
    }

    close(code?: number, reason?: string): void {
        this.socket.close(code, reason);
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