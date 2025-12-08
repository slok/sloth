package ui_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerServiceDetails(t *testing.T) {
	tests := map[string]struct {
		request    func() *http.Request
		mock       func(m mocks)
		expBody    []string
		expHeaders http.Header
		expCode    int
	}{
		"Listing the service details should redirect to the SLO listing page with the service ID as filter.": {
			request: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/u/app/services/svc-1", nil)
			},
			mock: func(m mocks) {},
			expHeaders: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
				"Location":     {"/u/app/slos?slo-service-id=svc-1"},
			},
			expCode: 303,
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
