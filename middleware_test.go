package mw

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var (
	middlewareOne = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("/mw1 before next"))
			next.ServeHTTP(w, r)
			w.Write([]byte("/mw1 after next"))
		})
	}

	middlewareTwo = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("/mw2 before next"))
			next.ServeHTTP(w, r)
			w.Write([]byte("/mw2 after next"))
		})
	}

	middlewareThree = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("/mw3 before next"))
			next.ServeHTTP(w, r)
			w.Write([]byte("/mw3 after next"))
		})
	}

	middlewareBreak Middleware = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("/skip the rest"))
		})
	}

	handlerOne = func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/first handler"))
	}

	handlerTwo = func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/second handler"))
	}

	handlerFinal = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/final handler"))
	})
)

func Test_Middleware(t *testing.T) {
	type testCase struct {
		title   string
		handler http.Handler
		out     string
	}

	cases := []testCase{
		testCase{
			title:   "build handler with single middleware (one call of Use() func with single argument)",
			handler: New().Use(middlewareOne).Then(handlerFinal),
			out:     "/mw1 before next/final handler/mw1 after next",
		},
		testCase{
			title:   "build handler with multiple middleware (adding one middleware per Use())",
			handler: New().Use(middlewareOne).Use(middlewareTwo).Use(middlewareThree).Then(handlerFinal),
			out:     "/mw1 before next/mw2 before next/mw3 before next/final handler/mw3 after next/mw2 after next/mw1 after next",
		},
		testCase{
			title:   "build handler with combination of single/plural calls of Use()",
			handler: New().Use(middlewareOne).Use(middlewareTwo, middlewareThree).Then(handlerFinal),
			out:     "/mw1 before next/mw2 before next/mw3 before next/final handler/mw3 after next/mw2 after next/mw1 after next",
		},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.handler.ServeHTTP(w, nil)
			if w.Body.String() != tc.out {
				t.Fatalf("The output %q is expected to be %q", w.Body.String(), tc.out)
			}
		})
	}
}

func Test_ChainAny(t *testing.T) {
	// replace blob handler in order to check if it is being called
	blobHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/blob handler"))
	}

	type testCase struct {
		title string
		args  []interface{}
		out   string
		err   error
	}

	cases := []testCase{
		testCase{
			title: "building handler with unsupported argument types should produce an error",
			args: []interface{}{
				middlewareOne,
				middlewareTwo,
				42,
				middlewareThree,
				handlerFinal,
			},
			err: errUnsupportedArgType(42),
		},
		testCase{
			title: "middleware should have control over the \"next\" handlers",
			args: []interface{}{
				middlewareOne,
				middlewareTwo,
				middlewareBreak,
				middlewareThree,
				handlerFinal,
			},
			out: "/mw1 before next/mw2 before next/skip the rest/mw2 after next/mw1 after next",
		},
		testCase{
			title: "calling function without any arguments should build a middleware with only blobHandler",
			out:   "/blob handler",
		},
		testCase{
			title: "building handler with all kind of supported arguments should be successful",
			args: []interface{}{
				middlewareOne,
				Middleware(middlewareTwo),
				handlerOne,
				http.HandlerFunc(handlerTwo),
				middlewareThree,
				handlerFinal,
			},
			out: "/mw1 before next/mw2 before next/first handler/second handler/mw3 before next/final handler/blob handler/mw3 after next/mw2 after next/mw1 after next",
		},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			h, err := Chain(tc.args...)
			if !reflect.DeepEqual(err, tc.err) {
				t.Fatalf("Error \"%v\" expected to be %v", err, tc.err)
			}
			// no sense to call handler if ChainAny returned an error
			if h != nil {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, nil)
				if w.Body.String() != tc.out {
					t.Fatalf("Out %v expected to be %v", w.Body.String(), tc.out)
				}
			}
		})
	}
}