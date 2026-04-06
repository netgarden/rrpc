import { RRPCClient, RRPCResponse, type RRPCError } from "$lib/rrpc";

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
        let reqBody : string | Uint8Array | null = null;

        const queryParams = new URLSearchParams();
        for (const [key, value] of Object.entries(request as Record<string, unknown>)) {
            if (value === null || value === undefined) continue;
            if (value instanceof Uint8Array) {
                queryParams.set(key, btoa(String.fromCharCode(...value)));
            } else {
                queryParams.set(key, String(value));
            }
        }
        reqUri = uri + '?' + queryParams.toString();

        const res : RRPCResponse = await this._client.sendRequest(method, reqUri, reqBody);
        if (res.error != null) {
            this.handleError(res.error);
        }

        const ret : EchoResponse = await res.response.json();

        return ret;
    }

}



class EchoRequest {
    message: string;
    count: number;
    flag: boolean;
}

class EchoResponse {
    result: string;
}
