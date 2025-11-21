package ui_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerIndex(t *testing.T) {
	tests := map[string]struct {
		request    func() *http.Request
		mock       func(m mocks)
		expBody    []string
		expHeaders http.Header
		expCode    int
	}{
		"Entering the index should redirect to the services selection page.": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u", nil)
			},
			mock: func(m mocks) {},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Location":     {"/u/app/services"},
			},
			expCode: 307,
			expBody: []string{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			m := newMocks(t)
			test.mock(m)

			h := newTestUIHandler(t, m)

			w := httptest.NewRecorder()
			h.ServeHTTP(w, test.request())

			assert.Equal(test.expCode, w.Code)
			assert.Equal(test.expHeaders, w.Header())
			assertContainsHTTPResponseBody(t, test.expBody, w)
		})

	}
}
