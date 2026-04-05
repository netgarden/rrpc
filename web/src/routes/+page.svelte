<script lang="ts">
    import { AuthClient } from '$lib/auth';
    import { TestClient } from '$lib/test';
    import { FilesClient } from '$lib/files';

    const apiUrl = '';

    const authClient = new AuthClient(apiUrl);
    const testClient = new TestClient(apiUrl);
    const filesClient = new FilesClient(apiUrl);

    let username = $state('');
    let password = $state('');
    let loginResult = $state<string | null>(null);
    let loginError = $state<string | null>(null);

    let pingResult = $state<string | null>(null);
    let pingError = $state<string | null>(null);

    let uploadFile = $state<File | null>(null);
    let uploadResult = $state<string | null>(null);
    let uploadError = $state<string | null>(null);

    let downloadFilename = $state('');
    let downloadResult = $state<string | null>(null);
    let downloadError = $state<string | null>(null);

    async function login() {
        loginResult = null;
        loginError = null;
        try {
            const res = await authClient.authService.login({ username, password });
            loginResult = res.ok ? 'Login successful' : 'Login failed';
        } catch (e) {
            loginError = String(e);
        }
    }

    async function logout() {
        loginResult = null;
        loginError = null;
        try {
            await authClient.authService.logout();
            loginResult = 'Logged out';
        } catch (e) {
            loginError = String(e);
        }
    }

    async function ping() {
        pingResult = null;
        pingError = null;
        try {
            await testClient.pingerService.ping();
            pingResult = 'Pong!';
        } catch (e) {
            pingError = String(e);
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
            uploadError = String(e);
        }
    }

    async function download() {
        downloadResult = null;
        downloadError = null;
        if (!downloadFilename.trim()) return;
        try {
            const content = await filesClient.filesService.download(downloadFilename.trim());
            downloadResult = content;
        } catch (e) {
            downloadError = String(e);
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