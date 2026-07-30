package httpx

import (
	"net/http"
	"reflect"

	"github.com/casara/arnon/openapi"
)

type endpointHandler struct {
	handler http.HandlerFunc

	operation *openapi.Operation

	requestType reflect.Type

	responseType reflect.Type
}

func (handler *endpointHandler) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	handler.handler(
		writer,
		request,
	)
}

func (handler *endpointHandler) OpenAPIOperation() *openapi.Operation {
	return handler.operation
}

func (handler *endpointHandler) RequestType() reflect.Type {
	return handler.requestType
}

func (handler *endpointHandler) ResponseType() reflect.Type {
	return handler.responseType
}
