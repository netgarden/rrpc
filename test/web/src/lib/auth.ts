import { RRPCClient, RRPCResponse } from "$lib/rrpc";

export class AuthClient {

    _client: RRPCClient;

    authService: AuthServiceHandler;
    

    constructor(url: string) {
        this._client = new RRPCClient(url);

        this.authService = new AuthServiceHandler(this._client);
        
    }

}


class AuthServiceHandler {

    _name: string = "Auth";
    _path: string = "Auth";

    _client: RRPCClient;

    constructor(client: RRPCClient) {
        this._client = client;
    }

    async login(request: LoginRequest) : Promise<LoginResponse> {

        const method = "POST";
        const methodPath = "login";
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        let reqUri : string = uri;
        let reqBody : string | Uint8Array | FormData | null = null;

        reqBody = JSON.stringify(request);

        const res : RRPCResponse = await this._client.sendRequest(method, reqUri, reqBody, "application/json");

        const ret : LoginResponse = await res.response.json();

        return ret;
    }

    async logout() : Promise<void> {

        const method = "POST";
        const methodPath = "logout";
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        let reqUri : string = uri;
        let reqBody : string | Uint8Array | FormData | null = null;

        const res : RRPCResponse = await this._client.sendRequest(method, reqUri, reqBody, "");

        return;
    }

}

class LoginRequest {
    username!: string;
    password!: string;
}

class LoginResponse {
    ok!: boolean;
}
