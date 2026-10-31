// Package apiwithexamples rebuilds learn.openapis.org
// examples/v3.0/api-with-examples.json with the public fizz API.
package apiwithexamples

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// The example payloads of the official document.
const (
	versionsList = `{
	"versions": [
		{
			"id": "v2.0",
			"links": [
				{
					"href": "http://127.0.0.1:8774/v2/",
					"rel": "self"
				}
			],
			"status": "CURRENT",
			"updated": "2011-01-21T11:33:21Z"
		},
		{
			"id": "v3.0",
			"links": [
				{
					"href": "http://127.0.0.1:8774/v3/",
					"rel": "self"
				}
			],
			"status": "EXPERIMENTAL",
			"updated": "2013-07-23T11:33:21Z"
		}
	]
}`
	versionsList300 = `{
	"versions": [
		{
			"id": "v2.0",
			"links": [
				{
					"href": "http://127.0.0.1:8774/v2/",
					"rel": "self"
				}
			],
			"status": "CURRENT",
			"updated": "2011-01-21T11:33:21Z"
		},
		{
			"id": "v3.0",
			"links": [
				{
					"href": "http://127.0.0.1:8774/v3/",
					"rel": "self"
				}
			],
			"status": "EXPERIMENTAL",
			"updated": "2013-07-23T11:33:21Z"
		}
	]
}`
	versionDetails200 = `{
	"version": {
		"id": "v2.0",
		"links": [
			{
				"href": "http://127.0.0.1:8774/v2/",
				"rel": "self"
			},
			{
				"href": "http://docs.openstack.org/api/openstack-compute/2/os-compute-devguide-2.pdf",
				"rel": "describedby",
				"type": "application/pdf"
			},
			{
				"href": "http://docs.openstack.org/api/openstack-compute/2/wadl/os-compute-2.wadl",
				"rel": "describedby",
				"type": "application/vnd.sun.wadl+xml"
			},
			{
				"href": "http://docs.openstack.org/api/openstack-compute/2/wadl/os-compute-2.wadl",
				"rel": "describedby",
				"type": "application/vnd.sun.wadl+xml"
			}
		],
		"media-types": [
			{
				"base": "application/xml",
				"type": "application/vnd.openstack.compute+xml;version=2"
			},
			{
				"base": "application/json",
				"type": "application/vnd.openstack.compute+json;version=2"
			}
		],
		"status": "CURRENT",
		"updated": "2011-01-21T11:33:21Z"
	}
}`
	versionDetails203 = `{
	"version": {
		"id": "v2.0",
		"links": [
			{
				"href": "http://23.253.228.211:8774/v2/",
				"rel": "self"
			},
			{
				"href": "http://docs.openstack.org/api/openstack-compute/2/os-compute-devguide-2.pdf",
				"rel": "describedby",
				"type": "application/pdf"
			},
			{
				"href": "http://docs.openstack.org/api/openstack-compute/2/wadl/os-compute-2.wadl",
				"rel": "describedby",
				"type": "application/vnd.sun.wadl+xml"
			}
		],
		"media-types": [
			{
				"base": "application/xml",
				"type": "application/vnd.openstack.compute+xml;version=2"
			},
			{
				"base": "application/json",
				"type": "application/vnd.openstack.compute+json;version=2"
			}
		],
		"status": "CURRENT",
		"updated": "2011-01-21T11:33:21Z"
	}
}`
)

// examples wraps the JSON payload in the one-entry examples map of a response.
func examples(payload string) map[string]any {
	var v any
	if err := json.Unmarshal([]byte(payload), &v); err != nil {
		panic(err)
	}
	return map[string]any{"foo": v}
}

// New returns the app: a fizz whose spec is the api-with-examples example.
func New() *fizz.Fizz {
	f := fizz.New()
	f.Generator().SetInfo(&openapi.Info{Title: "Simple API overview", Version: "2.0.0"})

	f.GET("/", []fizz.OperationOption{
		fizz.ID("listVersionsv2"),
		fizz.Summary("List API versions"),
		fizz.StatusDescription("200 response"),
		fizz.DefaultResponseExamples(examples(versionsList)),
		fizz.ResponseWithExamples("300", "300 response", nil, nil, examples(versionsList300)),
	}, tonic.Handler(func(*gin.Context) error { return nil }, 200))

	f.GET("/v2", []fizz.OperationOption{
		fizz.ID("getVersionDetailsv2"),
		fizz.Summary("Show API version details"),
		fizz.StatusDescription("200 response"),
		fizz.DefaultResponseExamples(examples(versionDetails200)),
		fizz.ResponseWithExamples("203", "203 response", nil, nil, examples(versionDetails203)),
	}, tonic.Handler(func(*gin.Context) error { return nil }, 200))
	return f
}
