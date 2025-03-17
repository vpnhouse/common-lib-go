package xhttp

import (
	"bytes"
	"net/http"
	"strings"
)

type requestStringer struct {
	*http.Request
}

func (s *requestStringer) String() string {
	if s == nil || s.Request == nil {
		return ""
	}

	strBuff := bytes.NewBufferString("")
	strBuff.WriteString("RemoteAddress=" + s.Request.RemoteAddr + " ")
	strBuff.WriteString("Host=" + s.Request.Host + " ")
	strBuff.WriteString("Method=" + s.Request.Method + " ")
	if s.Request.URL != nil {
		strBuff.WriteString("URL=" + s.Request.URL.String() + " ")
	}
	for name, val := range s.Request.Header {
		strBuff.WriteString(name + "=[" + strings.Join(val, ",") + "] ")
	}

	return strBuff.String()
}

func RequestStringer(r *http.Request) *requestStringer {
	return &requestStringer{Request: r}
}
