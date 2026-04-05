import { RRPCClient, RRPCResponse, type RRPCError } from "$lib/rrpc";

export class FilesClient {

    _client: RRPCClient;

    filesService: FilesServiceHandler;
    

    constructor(url: string) {
        this._client = new RRPCClient(url);

        this.filesService = new FilesServiceHandler(this._client);
        
    }

}


class FilesServiceHandler {

    _name: string = "Files";
    _path: string = "/files";

    _client: RRPCClient;

    constructor(client: RRPCClient) {
        this._client = client;
    }

    async upload(filename: string, data: any) : Promise<void> {

        const method = "POST";
        const methodPath = "/upload/" + encodeURIComponent(filename);
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        const reqData = {
            'data': data,
        };

        const reqDataStr : string = JSON.stringify(reqData);

        const res : RRPCResponse = await this._client.sendRequest(method, uri, reqDataStr);

        return;
    }

    async download(filename: string, ) : Promise<any> {

        const method = "GET";
        const methodPath = "/download/" + encodeURIComponent(filename);
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        const res : RRPCResponse = await this._client.sendRequest(method, uri, null);
        const bytes : any = await res.response.text();

        return bytes;
    }

}




