{{- define "serviceClientHandler" }}
{{- $service := . }}
{{- $serviceHasMultipart := false }}
{{- range $_, $method := .Methods }}
    {{- if eq $method.ContentType "multipart/form-data" }}
        {{- $serviceHasMultipart = true }}
    {{- end }}
{{- end -}}
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
{{ $isMultipart := eq $method.ContentType "multipart/form-data" -}}
{{ $pathParams := methodPathParams $method.Path -}}
func (h *{{ firstLetterToUpper $service.Name }}ServiceClientHandler) {{ firstLetterToUpper $method.Name -}}
        (ctx context.Context
        {{- range $_, $p := $pathParams }}, {{ $p }} string{{ end -}}
        {{- range $_, $input := $method.Inputs -}}
        , {{ if $input.IsStruct }}data *{{ $input.Type }}{{ else if eq $input.Type "bytes" }}r0 io.Reader{{ end }}
        {{- end -}}
    )
    {{- if gt (len $method.Outputs) 0 }} (
        {{- if (index $method.Outputs 0).IsStruct -}}
            *{{ (index $method.Outputs 0).Type }}
        {{- else if eq (index $method.Outputs 0).Type "bytes" -}}
            *SizedReadCloser
        {{- else if eq (index $method.Outputs 0).Type "string" -}}
            string
        {{- end }}, error)
    {{- else -}}
        error
    {{- end }} {

    methodType := "{{ $method.Type }}"
    methodPath := {{ methodPathToGo $method.Path }}
    url := h.client.url + "/" + h.path + "/" + methodPath

    {{ if and (eq (len $method.Inputs) 1) (index $method.Inputs 0).IsStruct -}}
    jsonBody, err := json.Marshal(data)
    if err != nil {
        return {{ range $_, $_ := $method.Outputs }}{{ if (index $method.Outputs 0).IsStruct }}nil{{ else if eq (index $method.Outputs 0).Type "string" }}""{{ else }}nil{{ end }}, {{ end }}rrpc.ErrRrpcBadRequest.WithCausef("failed to marshal request: %w", err)
    }
    {{ end -}}

    {{ if gt (len $method.Outputs) 0 }}r{{ else }}_{{ end }}, err :=
    {{- if $isMultipart }} h.doMultipartRequest(ctx, methodType, url,
        {{- range $i, $input := $method.Inputs }}{{ if $input.IsStruct }} "{{ $input.Name }}", data,{{ end }}{{ if eq $input.Type "bytes" }} "{{ $input.Name }}", r0{{ end }}{{- end -}}
    )
    {{- else }} h.doHttpRequest(ctx, methodType, url, "{{ $method.ContentType }}",
        {{- if eq (len $method.Inputs) 1 }}
            {{- if (index $method.Inputs 0).IsStruct -}}
                bytes.NewBuffer(jsonBody)
            {{- else if eq (index $method.Inputs 0).Type "bytes" -}}
                r0
            {{- end -}}
        {{ else -}}
            nil
        {{- end -}}
    )
    {{- end }}
    if err != nil {
        return {{ range $_, $_ := $method.Outputs }}{{ if (index $method.Outputs 0).IsStruct }}nil{{ else if eq (index $method.Outputs 0).Type "string" }}""{{ else }}nil{{ end }}, {{ end }}err
    }

    {{ if eq (len $method.Outputs) 1 -}}
    {{ if (index $method.Outputs 0).IsStruct -}}
    out := &{{ (index $method.Outputs 0).Type }}{}
    defer r.Close()
    respBody, err := h.readAll(r)
    if err != nil {
        return nil, err
    }
    if err := json.Unmarshal(respBody, out); err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to unmarshal response: %w", err)
    }

    {{ else if eq (index $method.Outputs 0).Type "string" -}}
    defer r.Close()
    respBody, err := h.readAll(r)
    if err != nil {
        return "", err
    }

    {{ end -}}
    {{ end -}}

    return {{ if eq (len $method.Outputs) 1 }}
    {{- $output := (index $method.Outputs 0) -}}
    {{- if $output.IsStruct }}out{{ else if eq $output.Type "bytes" }}r{{ else if eq $output.Type "string" }}string(respBody){{ end }}, {{ end }}nil
}

{{ end }}

func (h *{{ firstLetterToUpper $service.Name }}ServiceClientHandler) doHttpRequest(ctx context.Context, method string, url string, contentType string, body io.Reader) (*SizedReadCloser, error) {

    req, err := http.NewRequest(method, url, body)
    if err != nil {
       	return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to create http request: %w", err)
    }

    if ctx != nil {
        req = req.WithContext(ctx)
    }

    if body != nil {
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

{{ if $serviceHasMultipart -}}
func (h *{{ firstLetterToUpper $service.Name }}ServiceClientHandler) doMultipartRequest(ctx context.Context, method string, url string, structField string, in interface{}, bytesField string, r io.Reader) (*SizedReadCloser, error) {

    var buf bytes.Buffer
    mw := multipart.NewWriter(&buf)

    jsonData, err := json.Marshal(in)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to marshal json part: %w", err)
    }
    fw, err := mw.CreateFormField(structField)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to create form field: %w", err)
    }
    if _, err := fw.Write(jsonData); err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to write form field: %w", err)
    }

    bw, err := mw.CreateFormFile(bytesField, bytesField)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to create form file: %w", err)
    }
    if _, err := io.Copy(bw, r); err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to write bytes part: %w", err)
    }
    mw.Close()

    req, err := http.NewRequest(method, url, &buf)
    if err != nil {
        return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to create http request: %w", err)
    }
    if ctx != nil {
        req = req.WithContext(ctx)
    }
    req.Header.Set("Content-Type", mw.FormDataContentType())
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
{{ end }}
func (h *{{ firstLetterToUpper $service.Name }}ServiceClientHandler) readAll(r io.Reader) ([]byte, error) {
    b, err := io.ReadAll(r)
    if err != nil {
        return nil, rrpc.ErrRrpcBadResponse.WithCausef("failed to read server response body: %w", err)
    }
    return b, nil
}

{{- end }}