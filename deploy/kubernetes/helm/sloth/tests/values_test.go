package tests

type msi = map[string]interface{}

func defaultValues() msi {
	return msi{}
}

func customValues() msi {
	return msi{
		"labels": msi{
			"label-from": "test",
		},

		"image": msi{
			"repository": "slok/sloth-test",
			"tag":        "v1.42.42",
		},

		"sloth": msi{
			"resyncInterval": "17m",
			"workers":        99,
			"labelSelector":  `x=y,z!=y`,
			"namespace":      "somens",
			"optimizedRules": false,
			"extraLabels": msi{
				"k1": "v1",
				"k2": "v2",
			},
		},

		"commonPlugins": msi{
			"enabled": true,
			"gitRepo": msi{
				"url":    "https://github.com/slok/sloth-test-common-sli-plugins",
				"branch": "main",
			},
		},

		"metrics": msi{
			"enabled":        true,
			"scrapeInterval": "45s",
			"prometheusLabels": msi{
				"kp1": "vp1",
				"kp2": "vp2",
			},
		},

		"customSloConfig": msi{
			"data": msi{
				"customKey": "customValue",
			},
		},

		"securityContext": msi{
			"pod": msi{
				"runAsNonRoot": true,
				"runAsGroup":   1000,
				"runAsUser":    100,
				"fsGroup":      100,
			},
			"container": msi{
				"allowPrivilegeEscalation": false,
			},
		},
	}
}
