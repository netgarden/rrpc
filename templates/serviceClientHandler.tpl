{{- define "serviceClientHandler" }}
{{- $service := .}}
func new{{ firstLetterToUpper .Name }}ServiceClientHandler(client *Client) *{{ firstLetterToUpper .Name }}ServiceClientHandler {
    return &{{ firstLetterToUpper .Name }}ServiceClientHandler{
        client: client,
        name: "{{ .Name }}",
        path: "{{ .Path }}",
    }
}

type {{ firstLetterToUpper .Name }}ServiceClientHandler struct {
    name string
    path string
    client *Client
}

{{ range $_, $method := .Methods -}}
func (h *{{ firstLetterToUpper $service.Name }}ServiceClientHandler) {{ firstLetterToUpper $method.Name -}}
        (ctx context.Context
        {{- range $_, $input := $method.Inputs -}}
        , {{ if $input.IsStruct }}data *{{ $input.Type }}{{ else if eq $input.Type "bytes" }}r0 *SizedReader{{ end }}
        {{- end -}}
    )
    {{- if gt (len $method.Outputs) 0 }} (
        {{- if (index $method.Outputs 0).IsStruct -}}
            *{{ (index $method.Outputs 0).Type }}
        {{- else if eq (index $method.Outputs 0).Type "bytes" -}}
            *SizedReadCloser
        {{- end }}, error)
    {{- else -}}
        error
    {{- end }} {

    methodType := "{{ $method.Type }}"
    methodPath := "{{ $method.Path }}"
    methodContentType := "{{ $method.ContentType }}"
    url := h.client.url + "/" + h.path + "/" + methodPath

    {{ if gt (len $method.Outputs) 0 }}r{{ else }}_{{ end }}, err := h.doHttpRequest(ctx, methodType, url, methodContentType,
        {{- if eq (len $method.Inputs) 1 }}
            {{- if (index $method.Inputs 0).IsStruct -}}
                data
            {{- else if eq (index $method.Inputs 0).Type "bytes" -}}
                r0
            {{- end -}}
        {{ else -}}
            nil
        {{- end -}}
    )
    if err != nil {
        return {{ range $_, $_ := $method.Outputs }}nil, {{ end }}err
    }

    {{ if and (eq (len $method.Outputs) 1) ((index $method.Outputs 0).IsStruct) -}}
    out := &{{ (index $method.Outputs 0).Type }}{}

    defer r.Close()

    respBody, err := h.readAll(r)
    if err != nil {
        return nil, err
    }

    err = json.Unmarshal(respBody, out)
    if err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to read unmarshal body: %w", err)
    }

    {{ end -}}

    return {{ if eq (len $method.Outputs) 1 }}{{ if (index $method.Outputs 0).IsStruct }}out{{ else if eq (index $method.Outputs 0).Type "bytes" }}r{{ end }}, {{ end }}nil
}

{{ end }}

func (h *{{ firstLetterToUpper $service.Name }}ServiceClientHandler) doHttpRequest(ctx context.Context, method string, url string, contentType string, in interface{}) (*SizedReadCloser, error) {

    var reqBodyBuffer io.Reader = nil

    if in != nil {

        reqBody, err := json.Marshal(in)
        if err != nil {
            return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to marshal json request: %w", err)
        }

        reqBodyBuffer = bytes.NewBuffer(reqBody)
    }

    req, err := http.NewRequest(method, url, reqBodyBuffer)
    if err != nil {
       	return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to create http request: %w", err)
    }

    if ctx != nil {
        req = req.WithContext(ctx)
    }

    if in != nil {
        req.Header.Set("Content-Type", contentType)
    }

    req.Header.Set("Accept", "application/json")

    resp, err := h.client.Do(req)
    if err != nil {
        return nil, rrpc.ErrRrpcRequestFailed.WithCause(err)
    }

    if resp.StatusCode != 200 {
        defer resp.Body.Close()

        respBody, err := h.readAll(resp.Body)
        if err != nil {
            return nil, err
        }

        var rpcErr rrpc.RRPCError
		if err := json.Unmarshal(respBody, &rpcErr); err != nil {
			return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to unmarshal server error: %w", err)
		}

		return nil, rpcErr.WithCauseString()
    }

    var ret *SizedReadCloser

    contentLengthStr := resp.Header.Get("Content-Length")
    if contentLengthStr != "" {

        contentLength, err := strconv.ParseInt(contentLengthStr, 10, 64)
        if err != nil {
            return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to parse Content-Length header: %w", err)
        }

        ret = NewSizedReadCloser(resp.Body, contentLength)
    }

    return ret, nil
}

func (h *{{ firstLetterToUpper $service.Name }}ServiceClientHandler) readAll(r io.Reader) ([]byte, error) {
    b, err := io.ReadAll(r)
    if err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to read server response body: %w", err)
    }
    return b, nil
}

{{- end }}