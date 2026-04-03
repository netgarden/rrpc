{{ define "errors" -}}

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

{{ range $_, $error := .Errors -}}
class {{ firstLetterToUpper $error.Name }} {
    constructor() {
        super("{{ $error.Message }}")
    }
}
{{ end }}

{{- end }}