package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSLOGroupLabelsIDMarshal(t *testing.T) {
	tests := map[string]struct {
		sloID  string
		labels map[string]string
		expID  string
	}{
		"Marshalling grouped labels into the SLO ID should be marshaled correctly.": {
			sloID: "test1",
			labels: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			expID: "test1:azE9djEsazI9djI=",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			id := SLOGroupLabelsIDMarshal(tc.sloID, tc.labels)
			assert.Equal(tc.expID, id)
		})
	}
}

func TestSLOGroupLabelsIDUnmarshal(t *testing.T) {
	tests := map[string]struct {
		id        string
		expSLOID  string
		expLabels map[string]string
		expErr    bool
	}{
		"A id with labels should return the information.": {
			id:       "test1:azE9djEsazI9djI=",
			expSLOID: "test1",
			expLabels: map[string]string{
				"k1": "v1",
				"k2": "v2",
			},
			expErr: false,
		},

		"A id without labels should return the information.": {
			id:       "test1",
			expSLOID: "test1",
			expErr:   false,
		},

		"A id with incorrect labels should fail.": {
			id:     "test1:dsadasdasdsa",
			expErr: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			sloID, labels, err := SLOGroupLabelsIDUnmarshal(test.id)
			if test.expErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(test.expSLOID, sloID)
				assert.Equal(test.expLabels, labels)
			}
		})
	}
}
