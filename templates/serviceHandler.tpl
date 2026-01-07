{{ define "serviceHandler" -}}
{{ $service := . -}}
func new{{ firstLetterToUpper .Name }}ServiceHandler(service {{ firstLetterToUpper .Name }}Service) *{{ firstLetterToUpper .Name }}ServiceHandler {

    serviceHandler := &{{ firstLetterToUpper .Name }}ServiceHandler{}
    serviceHandler.name = "{{ .Name }}"
    serviceHandler.path = "{{ .Path }}"
    serviceHandler.methods = []MethodHolder{
        {{- range $_, $method := .Methods}}
        {
            Path:        "{{ $method.Path }}",
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
    path    string
    methods []MethodHolder

    service {{ firstLetterToUpper .Name }}Service
}

func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) Name() string {
    return h.name
}

func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) Path() string {
    return h.path
}

func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) Methods() []MethodHolder {
    return h.methods
}

{{ range $_, $method := .Methods -}}
func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) handle{{ firstLetterToUpper $method.Name }}(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
    ctx = context.WithValue(ctx, MethodNameCtxKey, "{{ $method.Name }}")

	{{ if and (eq (len $method.Inputs) 1) ((index $method.Inputs 0).IsStruct) -}}
	reqBody, err := h.readBody(r)
	if err != nil {
		return err
	}

	reqPayload := &{{ (index $method.Inputs 0).Type }}{}
	if err := json.Unmarshal(reqBody, reqPayload); err != nil {
		return ErrRrpcBadRequest.WithCausef("failed to unmarshal request data: %w", err)
	}

	{{ end -}}

    {{ if and (eq (len $method.Outputs) 1) (eq (index $method.Outputs 0).Type "bytes") -}}
    w1 := NewResponseWriter(w, "application/octet-stream")

    {{ end -}}

	// Call service method implementation.
    {{if and (eq (len $method.Outputs) 1) ((index $method.Outputs 0).IsStruct)}}ret0, {{end}}err1 := h.service.{{firstLetterToUpper $method.Name}}(ctx{{range $i, $input := $method.Inputs}}, {{if $input.IsStruct}}reqPayload{{else if eq $input.Type "bytes"}}r.Body{{end}}{{end}}{{if eq (len $method.Outputs) 1}}{{if eq (index $method.Outputs 0).Type "bytes"}}, w1{{end}}{{end}})
    if err1 != nil {
    	rpcErr, ok := err1.(RRPCError)
    	if !ok {
    		rpcErr = ErrRrpcEndpoint.WithCause(err1)
    	}
    	return rpcErr
    }

	// w.WriteHeader(http.StatusOK)

    {{ if eq (len $method.Outputs) 1 }}
    {{ if (index $method.Outputs 0).IsStruct }}
	respBody, err := json.Marshal(ret0)
	if err != nil {
		return ErrRrpcBadResponse.WithCausef("failed to marshal json response: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)

	{{ else if eq (index $method.Outputs 0).Type "bytes" }}
	w.Header().Set("Content-Type", "application/octet-stream")

	{{- end}}
	{{- end}}

	return nil
}

{{ end }}

func (h *{{ firstLetterToUpper $service.Name }}ServiceHandler) readBody(r *http.Request) ([]byte, error) {
    defer r.Body.Close()

    reqBody, err := io.ReadAll(r.Body)
	if err != nil {
	    return nil, ErrRrpcBadRequest.WithCausef("failed to read request data: %w", err)
	}

    return reqBody, nil
}

{{- end }}