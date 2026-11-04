package cais

import (
	"net/http"
	"slices"
)

type Middleware func(http.Handler) http.Handler

func applyChain(h http.Handler, mws ...Middleware) http.Handler {
	for _, mw := range slices.Backward(mws) {
		h = mw(h)
	}
	return h
}
