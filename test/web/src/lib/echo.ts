import { RRPCClient, RRPCResponse, RRPCStream, RRPCInStream, RRPCOutStream } from "$lib/rrpc";

export class EchoClient {

    _client: RRPCClient;

    echoService: EchoServiceHandler;
    

    constructor(url: string) {
        this._client = new RRPCClient(url);

        this.echoService = new EchoServiceHandler(this._client);
        
    }

}


class EchoServiceHandler {

    _name: string = "Echo";
    _path: string = "/echo";

    _client: RRPCClient;

    constructor(client: RRPCClient) {
        this._client = client;
    }

    async echo(request: EchoRequest) : Promise<EchoResponse> {

        const method = "GET";
        const methodPath = "Echo";
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        let reqUri : string = uri;
        let reqBody : string | Uint8Array | FormData | null = null;

        const queryParams = new URLSearchParams();
        for (const [key, value] of Object.entries(request as unknown as Record<string, unknown>)) {
            if (value === null || value === undefined) continue;
            if (value instanceof Uint8Array) {
                queryParams.set(key, btoa(String.fromCharCode(...value)));
            } else if (typeof value === 'object') {
                // A map-typed field (e.g. DatatableRequest.filters) — browsers
                // have no native nested-query-param representation, so encode
                // each entry as bracket notation ("key[subkey]=value"); the
                // server's DecodeQueryParams regroups these before unmarshaling.
                for (const [subkey, subvalue] of Object.entries(value as Record<string, unknown>)) {
                    if (subvalue === null || subvalue === undefined) continue;
                    queryParams.set(`${key}[${subkey}]`, String(subvalue));
                }
            } else {
                queryParams.set(key, String(value));
            }
        }
        reqUri = uri + '?' + queryParams.toString();

        const res : RRPCResponse = await this._client.sendRequest(method, reqUri, reqBody, "application/json");

        const ret : EchoResponse = await res.response.json();

        return ret;
    }

    async chat() : Promise<RRPCStream<ChatMessage, ChatReply>> {

        const methodPath = "Chat";
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        return this._client.openStream<ChatMessage, ChatReply>(uri);
    }

    async collect() : Promise<RRPCInStream<LogLine>> {

        const methodPath = "Collect";
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        return this._client.openInStream<LogLine>(uri);
    }

    async tick() : Promise<RRPCOutStream<TickResponse>> {

        const methodPath = "Tick";
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        return this._client.openOutStream<TickResponse>(uri);
    }

    async uploadBytes() : Promise<RRPCInStream<Uint8Array>> {

        const methodPath = "UploadBytes";
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        return this._client.openInStream<Uint8Array>(uri, true);
    }

}

class ChatMessage {
    text!: string;
}

class ChatReply {
    from!: string;
    text!: string;
}

class EchoRequest {
    message!: string;
    count!: number;
    flag!: boolean;
}

class EchoResponse {
    result!: string;
}

class LogLine {
    line!: string;
}

class TickResponse {
    n!: number;
}
