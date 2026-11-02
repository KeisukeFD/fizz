// Package link rebuilds learn.openapis.org examples/v3.0/link-example.json
// with the public fizz API.
package link

import (
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/KeisukeFD/fizz"
	"github.com/KeisukeFD/fizz/openapi"
)

// The schemas of the official document have lower-case names, and no
// required property.
type (
	user struct {
		Username string `json:"username,omitempty"`
		UUID     string `json:"uuid,omitempty"`
	}
	repository struct {
		Slug  string `json:"slug,omitempty"`
		Owner user   `json:"owner,omitempty"`
	}
	pullrequest struct {
		ID         int        `json:"id,omitempty"`
		Title      string     `json:"title,omitempty"`
		Repository repository `json:"repository,omitempty"`
		Author     user       `json:"author,omitempty"`
	}
)

type userIn struct {
	Username string `path:"username"`
}

type repositoryIn struct {
	Username string `path:"username"`
	Slug     string `path:"slug"`
}

type pullRequestsIn struct {
	Username string `path:"username"`
	Slug     string `path:"slug"`
	State    string `query:"state" enum:"open,merged,declined"`
}

type pullRequestIn struct {
	Username string `path:"username"`
	Slug     string `path:"slug"`
	PID      string `path:"pid"`
}

// withLink adds the link, a reference to the link component of the same
// name, to the 200 response of the operation. fizz has no option for links:
// CustomizeOperation is the documented way.
func withLink(name, component string) fizz.OperationOption {
	return fizz.CustomizeOperation(func(op *openapi.Operation) error {
		r := op.Responses["200"].Value
		r.Links = map[string]*openapi.LinkOrRef{
			name: {Ref: &openapi.Reference{Ref: "#/components/links/" + component}},
		}
		return nil
	})
}

// New returns the app: a fizz whose spec is the link example.
//
// The link components cannot be registered, as fizz has no API for them
// (see allow.json).
func New() *fizz.Fizz {
	f := fizz.New(
		fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }),
		fizz.WithPointersAreNullable(false),
	)
	g := f.Generator()
	// The official parameters are in declaration order, not sorted.
	g.SetSortParams(false)
	// The official integers carry no format; fizz writes int64 for int.
	if err := g.OverrideDataType(reflect.TypeFor[int](), "integer", ""); err != nil {
		panic(err)
	}
	g.SetInfo(&openapi.Info{Title: "Link Example", Version: "1.0.0"})

	f.GET("/2.0/users/:username", []fizz.OperationOption{
		fizz.ID("getUserByName"),
		fizz.StatusDescription("The User"),
		withLink("userRepositories", "UserRepositories"),
	}, tonic.Handler(func(*gin.Context, *userIn) (*user, error) { return nil, nil }, 200))

	f.GET("/2.0/repositories/:username", []fizz.OperationOption{
		fizz.ID("getRepositoriesByOwner"),
		fizz.StatusDescription("repositories owned by the supplied user"),
		withLink("userRepository", "UserRepository"),
	}, tonic.Handler(func(*gin.Context, *userIn) ([]repository, error) { return nil, nil }, 200))

	f.GET("/2.0/repositories/:username/:slug", []fizz.OperationOption{
		fizz.ID("getRepository"),
		fizz.StatusDescription("The repository"),
		withLink("repositoryPullRequests", "RepositoryPullRequests"),
	}, tonic.Handler(func(*gin.Context, *repositoryIn) (*repository, error) { return nil, nil }, 200))

	f.GET("/2.0/repositories/:username/:slug/pullrequests", []fizz.OperationOption{
		fizz.ID("getPullRequestsByRepository"),
		fizz.StatusDescription("an array of pull request objects"),
	}, tonic.Handler(func(*gin.Context, *pullRequestsIn) ([]pullrequest, error) { return nil, nil }, 200))

	f.GET("/2.0/repositories/:username/:slug/pullrequests/:pid", []fizz.OperationOption{
		fizz.ID("getPullRequestsById"),
		fizz.StatusDescription("a pull request object"),
		withLink("pullRequestMerge", "PullRequestMerge"),
	}, tonic.Handler(func(*gin.Context, *pullRequestIn) (*pullrequest, error) { return nil, nil }, 200))

	f.POST("/2.0/repositories/:username/:slug/pullrequests/:pid/merge", []fizz.OperationOption{
		fizz.ID("mergePullRequest"),
		fizz.StatusDescription("the PR was successfully merged"),
	}, tonic.Handler(func(*gin.Context, *pullRequestIn) error { return nil }, 204))
	return f
}
