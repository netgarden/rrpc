<script lang="ts">
    import { AuthClient } from '$lib/auth';
    import { TestClient } from '$lib/test';
    import { FilesClient } from '$lib/files';
    import { EchoClient } from '$lib/echo';
    import { RRPCError } from '$lib/rrpc';

    function errMsg(e: unknown): string {
        if (e instanceof Error || e instanceof RRPCError) {
            return e.message;
        }
        return JSON.parse(JSON.stringify(e));
    }

    const apiUrl = '';

    const authClient = new AuthClient(apiUrl);
    const testClient = new TestClient(apiUrl);
    const filesClient = new FilesClient(apiUrl);
    const echoClient = new EchoClient(apiUrl);

    let username = $state('');
    let password = $state('');
    let loginResult = $state<string | null>(null);
    let loginError = $state<string | null>(null);

    let pingResult = $state<string | null>(null);
    let pingError = $state<string | null>(null);

    let uploadFile = $state<File | null>(null);
    let uploadResult = $state<string | null>(null);
    let uploadError = $state<string | null>(null);

    let upload2File = $state<File | null>(null);
    let upload2Filename = $state('');
    let upload2Result = $state<string | null>(null);
    let upload2Error = $state<string | null>(null);

    let downloadFilename = $state('');
    let downloadResult = $state<string | null>(null);
    let downloadError = $state<string | null>(null);

    let echoMessage = $state('hello');
    let echoCount = $state(3);
    let echoFlag = $state(true);
    let echoResult = $state<string | null>(null);
    let echoError = $state<string | null>(null);

    async function login() {
        loginResult = null;
        loginError = null;
        try {
            const res = await authClient.authService.login({ username, password });
            loginResult = res.ok ? 'Login successful' : 'Login failed';
        } catch (e) {
            loginError = errMsg(e);
        }
    }

    async function logout() {
        loginResult = null;
        loginError = null;
        try {
            await authClient.authService.logout();
            loginResult = 'Logged out';
        } catch (e) {
            loginError = errMsg(e);
        }
    }

    async function ping() {
        pingResult = null;
        pingError = null;
        try {
            await testClient.pingerService.ping();
            pingResult = 'Pong!';
        } catch (e) {
            pingError = errMsg(e);
        }
    }

    async function upload() {
        uploadResult = null;
        uploadError = null;
        if (!uploadFile) return;
        try {
            const arrayBuffer = await uploadFile.arrayBuffer();
            await filesClient.filesService.upload(uploadFile.name, new Uint8Array(arrayBuffer));
            uploadResult = `Uploaded "${uploadFile.name}"`;
        } catch (e) {
            uploadError = errMsg(e);
        }
    }

    async function echoSend() {
        echoResult = null;
        echoError = null;
        try {
            const res = await echoClient.echoService.echo({ message: echoMessage, count: echoCount, flag: echoFlag });
            echoResult = res.result;
        } catch (e) {
            echoError = errMsg(e);
        }
    }

    async function upload2() {
        upload2Result = null;
        upload2Error = null;
        if (!upload2File) return;
        try {
            const arrayBuffer = await upload2File.arrayBuffer();
            const filename = upload2Filename.trim() || upload2File.name;
            await filesClient.filesService.upload2({ filename }, new Uint8Array(arrayBuffer));
            upload2Result = `Uploaded "${filename}"`;
        } catch (e) {
            upload2Error = errMsg(e);
        }
    }

    async function download() {
        downloadResult = null;
        downloadError = null;
        if (!downloadFilename.trim()) return;
        try {
            const bytes = await filesClient.filesService.download(downloadFilename.trim());
            downloadResult = new TextDecoder().decode(bytes);
        } catch (e) {
            downloadError = errMsg(e);
        }
    }
</script>

<main>
    <h1>RRPC Test</h1>

    <section>
        <h2>Auth</h2>
        <div class="form">
            <input type="text" bind:value={username} placeholder="Username" />
            <input type="password" bind:value={password} placeholder="Password" />
            <div class="buttons">
                <button onclick={login}>Login</button>
                <button onclick={logout}>Logout</button>
            </div>
        </div>
        {#if loginResult}<p class="result">{loginResult}</p>{/if}
        {#if loginError}<p class="error">{loginError}</p>{/if}
    </section>

    <section>
        <h2>Pinger</h2>
        <button onclick={ping}>Ping</button>
        {#if pingResult}<p class="result">{pingResult}</p>{/if}
        {#if pingError}<p class="error">{pingError}</p>{/if}
    </section>

    <section>
        <h2>Files — Upload</h2>
        <div class="form">
            <input type="file" onchange={(e) => uploadFile = e.currentTarget.files?.[0] ?? null} />
            <button onclick={upload} disabled={!uploadFile}>Upload</button>
        </div>
        {#if uploadResult}<p class="result">{uploadResult}</p>{/if}
        {#if uploadError}<p class="error">{uploadError}</p>{/if}
    </section>

    <section>
        <h2>Echo (GET query params)</h2>
        <div class="form">
            <input type="text" bind:value={echoMessage} placeholder="Message" />
            <input type="number" bind:value={echoCount} placeholder="Count" />
            <label><input type="checkbox" bind:checked={echoFlag} /> Flag</label>
            <button onclick={echoSend}>Send</button>
        </div>
        {#if echoResult}<p class="result">{echoResult}</p>{/if}
        {#if echoError}<p class="error">{echoError}</p>{/if}
    </section>

    <section>
        <h2>Files — Upload2 (multipart)</h2>
        <div class="form">
            <input type="file" onchange={(e) => upload2File = e.currentTarget.files?.[0] ?? null} />
            <input type="text" bind:value={upload2Filename} placeholder="Filename (optional, defaults to file name)" />
            <button onclick={upload2} disabled={!upload2File}>Upload</button>
        </div>
        {#if upload2Result}<p class="result">{upload2Result}</p>{/if}
        {#if upload2Error}<p class="error">{upload2Error}</p>{/if}
    </section>

    <section>
        <h2>Files — Download</h2>
        <div class="form">
            <input type="text" bind:value={downloadFilename} placeholder="Filename" />
            <button onclick={download} disabled={!downloadFilename.trim()}>Download</button>
        </div>
        {#if downloadResult}<pre class="result">{downloadResult}</pre>{/if}
        {#if downloadError}<p class="error">{downloadError}</p>{/if}
    </section>
</main>

<style>
    main {
        max-width: 480px;
        margin: 2rem auto;
        font-family: sans-serif;
    }

    section {
        margin-bottom: 2rem;
        padding: 1rem;
        border: 1px solid #ddd;
        border-radius: 4px;
    }

    .form {
        display: flex;
        flex-direction: column;
        gap: 0.5rem;
    }

    input[type="text"],
    input[type="password"] {
        padding: 0.4rem;
        border: 1px solid #ccc;
        border-radius: 4px;
    }

    .buttons {
        display: flex;
        gap: 0.5rem;
    }

    button {
        padding: 0.4rem 1rem;
        cursor: pointer;
    }

    button:disabled {
        opacity: 0.5;
        cursor: default;
    }

    .result {
        color: green;
        margin-top: 0.5rem;
        white-space: pre-wrap;
        word-break: break-all;
    }

    .error {
        color: red;
        margin-top: 0.5rem;
    }
</style>