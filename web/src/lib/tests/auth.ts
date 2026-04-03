import { RRPCClient, RRPCResponse } from "$lib/rrpc";

export class AuthClient {

    _client: RRPCClient;

    auth: AuthServiceHandler;
    

    constructor(url: string) {
        this._client = new RRPCClient(url);

        this.auth = new AuthServiceHandler(this._client);
        
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

        let method = "POST";
        let path = "login"
        let uri = "/" + this._path + "/" + path;

        
        let reqData = {
            'request': request,
        };

        let reqDataStr : string = JSON.stringify(reqData);

        let res : RRPCResponse = await this._client.sendRequest(method, uri, reqDataStr);
        let obj : LoginResponse = await res.response.json();

        return obj;
    }

    async logout() : Promise<void> {

        let method = "POST";
        let path = "logout"
        let uri = "/" + this._path + "/" + path;

        let res : RRPCResponse = await this._client.sendRequest(method, uri, null);

        return;
    }

}



class LoginRequest {
    username: any;
    password: any;
    
}

class LoginResponse {
    ok: any;
    
}


class UnexpectedReturnCodeError extends Error {

    code : string;
    reason : string;

    constructor(code : string, reason : string) {
        super("Received unexpected return code: '" + code + "' with reason '" + reason + "'")
        this.code = code;
        this.reason = reason;
    }

}

class InternalServerError extends Error {
    constructor() {
        super("Remote internal server error")
    }
}

