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

        const res : RRPCResponse = await this._client.sendRequest(method, uri, null);

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

    async serveStatic(filepath: string, ) : Promise<any> {

        const method = "GET";
        const methodPath = "/" + encodeURIComponent(filepath);
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        const res : RRPCResponse = await this._client.sendRequest(method, uri, null);
        const bytes : any = await res.response.text();

        return bytes;
    }

}




