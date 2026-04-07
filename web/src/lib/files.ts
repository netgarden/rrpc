import { RRPCClient, RRPCResponse } from "$lib/rrpc";

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

    async upload(filename: string, data: string | Uint8Array) : Promise<void> {

        const method = "PUT";
        const methodPath = "/upload/" + encodeURIComponent(filename);
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        let reqUri : string = uri;
        let reqBody : string | Uint8Array | FormData | null = null;

        reqBody = data;

        const res : RRPCResponse = await this._client.sendRequest(method, reqUri, reqBody);

        return;
    }

    async upload2(info: UploadDataInfo, data: string | Uint8Array) : Promise<void> {

        const method = "POST";
        const methodPath = "/upload2";
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        let reqUri : string = uri;
        let reqBody : string | Uint8Array | FormData | null = null;

        const formData = new FormData();
        formData.append('info', JSON.stringify(info));
        formData.append('data', new Blob([data as unknown as BlobPart]));
        reqBody = formData;

        const res : RRPCResponse = await this._client.sendRequest(method, reqUri, reqBody);

        return;
    }

    async download(filename: string) : Promise<Uint8Array> {

        const method = "GET";
        const methodPath = "/download/" + encodeURIComponent(filename);
        const uri = ('/' + this._path + '/' + methodPath).replace(/\/+/g, '/');

        let reqUri : string = uri;
        let reqBody : string | Uint8Array | FormData | null = null;

        const res : RRPCResponse = await this._client.sendRequest(method, reqUri, reqBody);

        const ret : Uint8Array = new Uint8Array(await res.response.arrayBuffer());

        return ret;
    }

}

class UploadDataInfo {
    filename!: string;
}
