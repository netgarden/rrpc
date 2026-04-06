import { RRPCClient, RRPCResponse, type RRPCError } from "$lib/rrpc";

export class TestClient {

    _client: RRPCClient;

    pingerService: PingerServiceHandler;
    staticService: StaticServiceHandler;
    

    constructor(url: string) {
        this._client = new RRPCClient(url);

        this.pingerService = new PingerServiceHandler(this._client);
        this.staticService = new StaticServiceHandler(this._client);
        
    }

}


class PingerServiceHandler {

    _name: string = "Pinger";
    _path: string = "Pinger";

    _client: RRPCClient;

    constructor(client: RRPCClient) {
        this._client = client;
    }

    async ping() : Promise<void> {

        const method = "GET";
        const methodPath = "Ping";
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        let reqUri : string = uri;
        let reqBody : string | Uint8Array | null = null;

        const res : RRPCResponse = await this._client.sendRequest(method, reqUri, reqBody);
        if (res.error != null) {
            this.handleError(res.error);
        }

        return;
    }

}

class StaticServiceHandler {

    _name: string = "Static";
    _path: string = "/";

    _client: RRPCClient;

    constructor(client: RRPCClient) {
        this._client = client;
    }

    async serveStatic(filepath: string) : Promise<Uint8Array> {

        const method = "GET";
        const methodPath = "/" + encodeURIComponent(filepath);
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        let reqUri : string = uri;
        let reqBody : string | Uint8Array | null = null;

        const res : RRPCResponse = await this._client.sendRequest(method, reqUri, reqBody);
        if (res.error != null) {
            this.handleError(res.error);
        }

        const ret : Uint8Array = new Uint8Array(await res.response.arrayBuffer());

        return ret;
    }

}


