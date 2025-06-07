package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

func upgradeError(w http.ResponseWriter, err error) (Connection, error) {
	w.Header().Set("Connection", "Upgrade")
	w.Header().Set("Upgrade", "websocket")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte(err.Error()))
	return Connection{}, err
}

func Upgrade(w http.ResponseWriter, r *http.Request) (Connection, error) {
	if r.Method != "GET" {
		return upgradeError(w, errors.New("not a websocket upgrade request: method must be GET"))
	}

	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return upgradeError(w, errors.New("not a websocket upgrade request: missing or invalid 'upgrade' header"))
	}

	connectionHeader := r.Header.Get("Connection")
	connectionValues := strings.Split(connectionHeader, ",")
	hasUpgrade := false
	for _, value := range connectionValues {
		if strings.EqualFold(strings.TrimSpace(value), "upgrade") {
			hasUpgrade = true
			break
		}
	}
	if !hasUpgrade {
		return upgradeError(w, errors.New("not a websocket upgrade request: 'connection' header does not contain 'upgrade'"))
	}

	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		w.Header().Set("Sec-WebSocket-Version", "13")
		w.WriteHeader(http.StatusUpgradeRequired)
		_, _ = w.Write([]byte("websocket version not supported, please use version 13"))
		return Connection{}, errors.New("websocket version not supported")
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return upgradeError(w, errors.New("websocket key missing"))
	}

	h := sha1.New()
	h.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	acceptKey := base64.StdEncoding.EncodeToString(h.Sum(nil))

	hj, ok := w.(http.Hijacker)
	if !ok {
		return upgradeError(w, errors.New("connection doesn't support hijacking"))
	}

	conn, bufrw, err := hj.Hijack()
	if err != nil {
		return Connection{}, err
	}

	bufrw.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	bufrw.WriteString("Upgrade: websocket\r\n")
	bufrw.WriteString("Connection: Upgrade\r\n")
	bufrw.WriteString("Sec-WebSocket-Accept: " + acceptKey + "\r\n")

	if subprotocol := r.Header.Get("Sec-WebSocket-Protocol"); subprotocol != "" {
		protocols := strings.Split(subprotocol, ",")
		if len(protocols) > 0 {
			chosenProtocol := strings.TrimSpace(protocols[0])
			bufrw.WriteString("Sec-WebSocket-Protocol: " + chosenProtocol + "\r\n")
		}
	}

	bufrw.WriteString("\r\n")
	bufrw.Flush()

	return newConnection(conn, false), nil
}
