package readwriter

import (
	"fmt"
	"io"
	"net/http"

	"github.com/obscuronet/go-obscuro/go/common/log"

	"github.com/gorilla/websocket"
)

const (
	httpCodeErr = 500
)

var upgrader = websocket.Upgrader{} // Used to upgrade connections to websocket connections.

// ReadWriter handles reading and writing Ethereum JSON RPC requests.
type ReadWriter interface {
	ReadRequest() ([]byte, error)
	WriteResponse(msg []byte) error
	HandleError(msg string)
	SupportsSubscriptions() bool
}

// HTTPReadWriter is a ReadWriter over HTTP.
type HTTPReadWriter struct {
	resp http.ResponseWriter
	req  *http.Request
}

// WSReadWriter is a ReadWriter over websockets.
type WSReadWriter struct {
	conn *websocket.Conn
}

func NewHTTPReadWriter(resp http.ResponseWriter, req *http.Request) ReadWriter {
	return &HTTPReadWriter{resp: resp, req: req}
}

func NewWSReadWriter(resp http.ResponseWriter, req *http.Request) (ReadWriter, error) {
	// We search all the request's headers. If there's a websocket upgrade header, we upgrade to a websocket connection.
	conn, err := upgrader.Upgrade(resp, req, nil)
	if err != nil {
		err = fmt.Errorf("unable to upgrade to websocket connection")
		httpLogAndSendErr(resp, err.Error()) // TODO - Handle error properly for websockets.
		return nil, err
	}

	return &WSReadWriter{
		conn: conn,
	}, nil
}

func (h *HTTPReadWriter) ReadRequest() ([]byte, error) {
	body, err := io.ReadAll(h.req.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read request body: %w", err)
	}
	return body, nil
}

func (h *HTTPReadWriter) WriteResponse(msg []byte) error {
	_, err := h.resp.Write(msg)
	if err != nil {
		return fmt.Errorf("could not write response: %w", err)
	}
	return nil
}

func (h *HTTPReadWriter) HandleError(msg string) {
	httpLogAndSendErr(h.resp, msg)
}

func (h *HTTPReadWriter) SupportsSubscriptions() bool {
	return false
}

func (w *WSReadWriter) ReadRequest() ([]byte, error) {
	panic("not implemented")
}

func (w *WSReadWriter) WriteResponse(msg []byte) error {
	panic("not implemented")
}

func (w *WSReadWriter) HandleError(msg string) {
	panic("not implemented")
}

func (w *WSReadWriter) SupportsSubscriptions() bool {
	return true
}

// Logs the error, prints it to the console, and returns the error over HTTP.
func httpLogAndSendErr(resp http.ResponseWriter, msg string) {
	log.Error(msg)
	fmt.Println(msg)
	http.Error(resp, msg, httpCodeErr)
}