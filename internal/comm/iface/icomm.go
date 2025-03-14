package iface

import (
	"net/http"
	"sync"
)

type IComm interface {
	ApiUpdate(w http.ResponseWriter, r *http.Request)
	ApiRestart(w http.ResponseWriter, r *http.Request)
	ApiUninstall(w http.ResponseWriter, r *http.Request)
	ApiVersion(w http.ResponseWriter, r *http.Request)
	GetBuffer() *sync.Pool
}
