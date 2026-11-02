// Package uspto rebuilds learn.openapis.org examples/v3.0/uspto.json with
// the public fizz API.
package uspto

import (
	"encoding/json"
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// dataSetList is the list of the data sets.
type dataSetList struct {
	Apis []struct {
		APIDocumentationURL string `json:"apiDocumentationUrl,omitempty" format:"uri-reference" description:"A URL to the API console for each API"`
		APIKey              string `json:"apiKey,omitempty" description:"To be used as a dataset parameter value"`
		APIURL              string `json:"apiUrl,omitempty" format:"uri-reference" description:"The URL describing the dataset's fields"`
		APIVersionNumber    string `json:"apiVersionNumber,omitempty" description:"To be used as a version parameter value"`
	} `json:"apis,omitempty"`
	Total int `json:"total,omitempty"`
}

// fieldsIn is the input of the fields operation.
type fieldsIn struct {
	Dataset string `path:"dataset" description:"Name of the dataset."`
	Version string `path:"version" description:"Version of the dataset."`
}

// searchIn is the input of the search operation. Its body is the form
// described by searchForm.
type searchIn struct {
	Version string `path:"version" default:"v1" description:"Version of the dataset."`
	Dataset string `path:"dataset" default:"oa_citations" description:"Name of the dataset. In this case, the default value is oa_citations"`
}

// searchForm is the form body of the search operation. Its type is
// anonymous, so that the schema is inlined as in the official document.
var searchForm = struct {
	Criteria string `json:"criteria" validate:"required" default:"*:*" description:"Uses Lucene Query Syntax in the format of propertyName:value, propertyName:[num1 TO num2] and date range format: propertyName:[yyyyMMdd TO yyyyMMdd]. In the response please see the 'docs' element which has the list of record objects. Each record structure would consist of all the fields and their corresponding values."`
	Rows     int    `json:"rows,omitempty" default:"100" description:"Specify number of rows to be returned. If you run the search with default values, in the response you will see 'numFound' attribute which will tell the number of records available in the dataset."`
	Start    int    `json:"start,omitempty" default:"0" description:"Starting record number. Default value is 0."`
}{}

const listExample = `{
	"apis": [
		{
			"apiDocumentationUrl": "https://developer.uspto.gov/ds-api-docs/index.html?url=https://developer.uspto.gov/ds-api/swagger/docs/oa_citations.json",
			"apiKey": "oa_citations",
			"apiUrl": "https://developer.uspto.gov/ds-api/oa_citations/v1/fields",
			"apiVersionNumber": "v1"
		},
		{
			"apiDocumentationUrl": "https://developer.uspto.gov/ds-api-docs/index.html?url=https://developer.uspto.gov/ds-api/swagger/docs/cancer_moonshot.json",
			"apiKey": "cancer_moonshot",
			"apiUrl": "https://developer.uspto.gov/ds-api/cancer_moonshot/v1/fields",
			"apiVersionNumber": "v1"
		}
	],
	"total": 2
}`

// New returns the app: a fizz whose spec is the uspto example.
func New() *fizz.Fizz {
	f := fizz.New(fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }))
	g := f.Generator()
	// The official parameters are in declaration order, not sorted.
	g.SetSortParams(false)
	// The official integers carry no format; fizz writes int64 for int.
	if err := g.OverrideDataType(reflect.TypeFor[int](), "integer", ""); err != nil {
		panic(err)
	}
	g.SetInfo(&openapi.Info{
		Title:       "USPTO Data Set API",
		Version:     "1.0.0",
		Description: "The Data Set API (DSAPI) allows the public users to discover and search USPTO exported data sets. This is a generic API that allows USPTO users to make any CSV based data files searchable through API. With the help of GET call, it returns the list of data fields that are searchable. With the help of POST call, data can be fetched based on the filters on the field names. Please note that POST call is used to search the actual data. The reason for the POST call is that it allows users to specify any complex search criteria without worry about the GET size limitations as well as encoding of the input parameters.",
		Contact: &openapi.Contact{
			Name:  "Open Data Portal",
			URL:   "https://developer.uspto.gov",
			Email: "developer@uspto.gov",
		},
	})
	g.SetServers([]*openapi.Server{{
		URL: "{scheme}://developer.uspto.gov/ds-api",
		Variables: map[string]*openapi.ServerVariable{
			"scheme": {
				Default:     "https",
				Enum:        []string{"https", "http"},
				Description: "The Data Set API is accessible via https and http",
			},
		},
	}})
	g.SetTag(&openapi.Tag{Name: "metadata", Description: "Find out about the data sets"})
	g.SetTag(&openapi.Tag{Name: "search", Description: "Search a data set"})

	var list any
	if err := json.Unmarshal([]byte(listExample), &list); err != nil {
		panic(err)
	}

	f.GET("/", []fizz.OperationOption{
		fizz.ID("list-data-sets"),
		fizz.Summary("List available data sets"),
		fizz.Tags("metadata"),
		fizz.StatusDescription("Returns a list of data sets"),
		fizz.DefaultResponseExamples(map[string]any{"list": list}),
	}, tonic.Handler(func(*gin.Context) (*dataSetList, error) { return nil, nil }, 200))

	f.GET("/:dataset/:version/fields", []fizz.OperationOption{
		fizz.ID("list-searchable-fields"),
		fizz.Summary("Provides the general information about the API and the list of fields that can be used to query the dataset."),
		fizz.Description("This GET API returns the list of all the searchable field names that are in the oa_citations. Please see the 'fields' attribute which returns an array of field names. Each field or a combination of fields can be searched using the syntax options shown below."),
		fizz.Tags("metadata"),
		fizz.StatusDescription("The dataset API for the given version is found and it is accessible to consume."),
		fizz.Parameter(&openapi.Parameter{Name: "dataset", In: "path", Example: "oa_citations"}),
		fizz.Parameter(&openapi.Parameter{Name: "version", In: "path", Example: "v1"}),
		fizz.Response("404", "The combination of dataset name and version is not found in the system or it is not published yet to be consumed by public.", "", nil, nil),
	}, tonic.Handler(func(*gin.Context, *fieldsIn) (string, error) { return "", nil }, 200))

	f.POST("/:dataset/:version/records", []fizz.OperationOption{
		fizz.ID("perform-search"),
		fizz.Summary("Provides search capability for the data set with the given search criteria."),
		fizz.Description("This API is based on Solr/Lucene Search. The data is indexed using SOLR. This GET API returns the list of all the searchable field names that are in the Solr Index. Please see the 'fields' attribute which returns an array of field names. Each field or a combination of fields can be searched using the Solr/Lucene Syntax. Please refer https://lucene.apache.org/core/3_6_2/queryparsersyntax.html#Overview for the query syntax. List of field names that are searchable can be determined using above GET api."),
		fizz.Tags("search"),
		fizz.StatusDescription("successful operation"),
		fizz.RequestBody("", false, fizz.Content("application/x-www-form-urlencoded", searchForm)),
		fizz.Response("404", "No matching record found for the given criteria.", nil, nil, nil),
	}, tonic.Handler(func(*gin.Context, *searchIn) ([]map[string]struct{}, error) { return nil, nil }, 200))
	return f
}
