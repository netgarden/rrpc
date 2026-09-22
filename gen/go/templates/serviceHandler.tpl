{{ define "serviceHandler" -}}
{{ $service := . -}}
func new{{ firstLetterToUpper .Name }}ServiceHandler(service {{ firstLetterToUpper .Name }}Service) *{{ firstLetterToUpper .Name }}ServiceHandler {

    serviceHandler := &{{ firstLetterToUpper .Name }}ServiceHandler{}
    serviceHandler.name = "{{ .Name }}"
    serviceHandler.path = rrpc.ParsePath("{{ .Path }}")
    serviceHandler.methods = []rrpc.MethodHolder{
        {{- range $_, $method := .Methods}}
        {
            Path:        rrpc.ParsePath("{{ $method.Path }}"),
            Type:        "{{ $method.Type }}",
            ContentType: "{{ $method.ContentType }}",
            HandlerFunc: serviceHandler.handle{{ firstLetterToUpper $method.Name }},
        },
        {{- end}}
    }
    serviceHandler.service = service

    return serviceHandler
}

type {{ firstLetterToUpper .Name }}ServiceHandler struct {
    name    string
    path    *rrpc.Path
    methods []rrpc.MethodHolder

    service {{ firstLetterToUpper .Name }}Service
}

func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) Name() string {
    return h.name
}

func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) Path() *rrpc.Path {
    return h.path
}

func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) Methods() []rrpc.MethodHolder {
    return h.methods
}

{{ range $_, $method := .Methods -}}
{{ if $method.Streaming -}}
func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) handle{{ firstLetterToUpper $method.Name }}(ctx *rrpc.Context) error {
    stream, err := rrpc.{{ streamUpgradeCall $method }}(ctx, rrpc.UpgradeOptions{})
    if err != nil {
        return err
    }
    defer stream.CloseGracefully()
    return h.service.{{ firstLetterToUpper $method.Name }}(ctx, stream)
}

{{ else -}}
func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) handle{{ firstLetterToUpper $method.Name }}(ctx *rrpc.Context) error {
    {{- $isMultipart := eq $method.ContentType "multipart/form-data" }}

	{{ if $isMultipart -}}
	if err := ctx.Request().ParseMultipartForm(32 << 20); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to parse multipart form: %w", err)
	}
	{{- range $i, $input := $method.Inputs }}
	{{- if $input.IsStruct }}
	reqPayload := &{{ $input.Type }}{}
	if err := json.Unmarshal([]byte(ctx.Request().FormValue("{{ $input.Name }}")), reqPayload); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to unmarshal {{ $input.Name }} field: %w", err)
	}
	{{- end }}
	{{- if eq $input.Type "bytes" }}
	{{ $input.Name }}File, _, err := ctx.Request().FormFile("{{ $input.Name }}")
	if err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to read {{ $input.Name }} field: %w", err)
	}
	defer {{ $input.Name }}File.Close()
	{{- end }}
	{{- end }}

	{{ else if and (eq (len $method.Inputs) 1) ((index $method.Inputs 0).IsStruct) -}}
	reqPayload := &{{ (index $method.Inputs 0).Type }}{}
	{{ if eq $method.Type "GET" -}}
	if err := rrpc.DecodeQueryParams(ctx.Request().URL.Query(), reqPayload); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to decode query params: %w", err)
	}
	{{ else -}}
	reqBody, err := h.readBody(ctx.Request())
	if err != nil {
		return err
	}
	if err := json.Unmarshal(reqBody, reqPayload); err != nil {
		return rrpc.ErrRrpcBadRequest.WithCausef("failed to unmarshal request data: %w", err)
	}
	{{ end -}}
	{{ end -}}

    {{ if and (eq (len $method.Outputs) 1) (eq (index $method.Outputs 0).Type "bytes") -}}
    w1 := rrpc.NewResponseWriter(ctx.Response(), "{{ $method.ResponseContentType }}")

    {{ end -}}

	// Call service method implementation.
    {{if and (eq (len $method.Outputs) 1) ((index $method.Outputs 0).IsStruct)}}ret0, {{end}}err1 := h.service.{{firstLetterToUpper $method.Name}}(ctx{{range $i, $input := $method.Inputs}}, {{if $input.IsStruct}}reqPayload{{else if eq $input.Type "bytes"}}{{if $isMultipart}}{{ $input.Name }}File{{else}}ctx.Request().Body{{end}}{{end}}{{end}}{{if eq (len $method.Outputs) 1}}{{if eq (index $method.Outputs 0).Type "bytes"}}, w1{{end}}{{end}})
    if err1 != nil {
    	rpcErr, ok := err1.(rrpc.RRPCError)
    	if !ok {
    		rpcErr = rrpc.ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// ctx.Response().WriteHeader(http.StatusOK)

    {{ if eq (len $method.Outputs) 1 }}
    {{ if (index $method.Outputs 0).IsStruct }}
	respBody, err := json.Marshal(ret0)
	if err != nil {
		return rrpc.ErrRrpcBadResponse.WithCausef("failed to marshal json response: %w", err)
	}

    ctx.Response().Header().Set("Content-Type", "application/json")
    ctx.Response().Write(respBody)

	{{ else if eq (index $method.Outputs 0).Type "bytes" }}
        if ctx.Response().Header().Get("Content-Type") == "" {
            ctx.Response().Header().Set("Content-Type", "{{ $method.ResponseContentType }}")
        }

	{{- end}}
	{{- end}}

	return nil
}

{{ end -}}
{{ end }}

func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) readBody(r *http.Request) ([]byte, error) {
    defer r.Body.Close()

    reqBody, err := io.ReadAll(r.Body)
	if err != nil {
	    return nil, rrpc.ErrRrpcBadRequest.WithCausef("failed to read request data: %w", err)
	}

    return reqBody, nil
}

{{- end }}