// Package tictactoe rebuilds learn.openapis.org examples/v3.1/tictactoe.json
// with the public fizz API.
package tictactoe

import (
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/tonic"

	"github.com/wI2L/fizz"
	"github.com/wI2L/fizz/openapi"
)

// The schemas of the official document have lower-case names.

// coordinate is a row or a column of the board.
type coordinate int

// CustomizeSchema bounds the coordinate.
func (coordinate) CustomizeSchema(s *openapi.Schema) error {
	s.Minimum = "1"
	s.Maximum = "3"
	s.Examples = []any{1}
	return nil
}

// mark is the content of a square.
type mark string

// CustomizeSchema lists the marks.
func (mark) CustomizeSchema(s *openapi.Schema) error {
	s.Description = "Possible values for a board square. `.` means empty square."
	s.Enum = []any{".", "X", "O"}
	s.Examples = []any{"."}
	return nil
}

// winner is the winner of the game.
type winner string

// CustomizeSchema lists the winners.
func (winner) CustomizeSchema(s *openapi.Schema) error {
	s.Description = "Winner of the game. `.` means nobody has won yet."
	s.Enum = []any{".", "X", "O"}
	s.Examples = []any{"."}
	return nil
}

// errorMessage describes an error.
type errorMessage string

// CustomizeSchema describes and bounds the message.
func (errorMessage) CustomizeSchema(s *openapi.Schema) error {
	s.Description = "A text message describing an error"
	s.MaxLength = new(uint64(256))
	return nil
}

// board is the 3x3 grid of marks.
type board [3][3]mark

// CustomizeSchema fixes the size of the grid.
func (board) CustomizeSchema(s *openapi.Schema) error {
	s.MinItems, s.MaxItems = new(uint64(3)), new(uint64(3))
	s.Items.MinItems, s.Items.MaxItems = new(uint64(3)), new(uint64(3))
	return nil
}

// status is the state of the game.
type status struct {
	Winner winner `json:"winner,omitempty"`
	Board  board  `json:"board,omitempty"`
}

type squareIn struct {
	Row    coordinate `path:"row" description:"Board row (vertical coordinate)"`
	Column coordinate `path:"column" description:"Board column (horizontal coordinate)"`
}

type markIn struct {
	Row         coordinate `path:"row" description:"Board row (vertical coordinate)"`
	Column      coordinate `path:"column" description:"Board column (horizontal coordinate)"`
	ProgressURL string     `header:"progressUrl" description:"Progress URL that should be called if asynchronous response is returned"`
}

// requirement is a security requirement of an operation.
func requirement(scheme string, scopes ...string) fizz.OperationOption {
	if scopes == nil {
		scopes = []string{}
	}
	return fizz.Security(&openapi.SecurityRequirement{scheme: scopes})
}

// markSquareLink adds the markSquare link to the 200 response of the
// operation. fizz has no option for links: CustomizeOperation is the
// documented way.
func markSquareLink(op *openapi.Operation) error {
	op.Responses["200"].Value.Links = map[string]*openapi.LinkOrRef{
		"markSquare": {Value: &openapi.Link{
			Description: "Operation to use if the mark value returned indicates the square is empty",
			OperationID: "put-square",
			Parameters: map[string]any{
				"column": "$request.path.column",
				"row":    "$request.path.row",
			},
		}},
	}
	return nil
}

// statusCallback adds the statusCallback callback to the operation. fizz
// has no option for callbacks: CustomizeOperation is the documented way.
func statusCallback(op *openapi.Operation) error {
	op.Callbacks = map[string]*openapi.CallbackOrRef{
		"statusCallback": {Value: &openapi.Callback{Expressions: map[string]*openapi.PathItem{
			"{$request.header.progressUrl}": {POST: &openapi.Operation{
				Summary:     "Status of mark operation",
				Description: "Provides the status of the mark operation",
				ID:          "markOperationCallback",
				RequestBody: &openapi.RequestBody{
					Content: map[string]*openapi.MediaType{
						"application/json": {Schema: &openapi.Schema{Ref: "#/components/schemas/status"}},
					},
				},
				Responses: openapi.Responses{
					"200": {Value: &openapi.Response{Description: "Mark operation status received"}},
				},
			}},
		}}},
	}
	return nil
}

// New returns the app: a fizz whose spec is the tictactoe example.
//
// The markOperationWebhook webhook is not reproduced, as fizz has no API
// for webhooks (see allow.json).
func New() *fizz.Fizz {
	f := fizz.New(
		fizz.WithSchemaNamer(func(t reflect.Type) string { return t.Name() }),
		fizz.WithPointersAreNullable(false),
	)
	g := f.Generator()
	// The official parameters are in declaration order, not sorted.
	g.SetSortParams(false)
	// The official integer carries no format; fizz writes int64 for it.
	if err := g.OverrideDataType(reflect.TypeFor[coordinate](), "integer", ""); err != nil {
		panic(err)
	}
	g.SetInfo(&openapi.Info{
		Title:       "Tic Tac Toe",
		Version:     "1.0.0",
		Description: "This API allows writing down marks on a Tic Tac Toe board\nand requesting the state of the board or of individual squares.\n",
	})
	g.SetTag(&openapi.Tag{Name: "Gameplay"})
	g.SetSecuritySchemes(map[string]*openapi.SecuritySchemeOrRef{
		"defaultApiKey": {Value: &openapi.SecurityScheme{
			Type:        "apiKey",
			Description: "API key provided in console",
			In:          "header",
			Name:        "api-key",
		}},
		"basicHttpAuthentication": {Value: &openapi.SecurityScheme{
			Type:        "http",
			Description: "Basic HTTP Authentication",
			Scheme:      "Basic",
		}},
		"bearerHttpAuthentication": {Value: &openapi.SecurityScheme{
			Type:         "http",
			Description:  "Bearer token using a JWT",
			Scheme:       "Bearer",
			BearerFormat: "JWT",
		}},
		"app2AppOauth": {Value: &openapi.SecurityScheme{
			Type: "oauth2",
			Flows: &openapi.OAuthFlows{ClientCredentials: &openapi.OAuthFlow{
				TokenURL: "https://learn.openapis.org/oauth/2.0/token",
				Scopes:   map[string]string{"board:read": "Read the board"},
			}},
		}},
		"user2AppOauth": {Value: &openapi.SecurityScheme{
			Type: "oauth2",
			Flows: &openapi.OAuthFlows{AuthorizationCode: &openapi.OAuthFlow{
				AuthorizationURL: "https://learn.openapis.org/oauth/2.0/auth",
				TokenURL:         "https://learn.openapis.org/oauth/2.0/token",
				Scopes: map[string]string{
					"board:read":  "Read the board",
					"board:write": "Write to the board",
				},
			}},
		}},
	})
	// The parameter components of the official document.
	for name, p := range map[string]*openapi.Parameter{
		"rowParam": {
			Name: "row", In: "path", Required: true,
			Description: "Board row (vertical coordinate)",
			Schema:      &openapi.Schema{Ref: "#/components/schemas/coordinate"},
		},
		"columnParam": {
			Name: "column", In: "path", Required: true,
			Description: "Board column (horizontal coordinate)",
			Schema:      &openapi.Schema{Ref: "#/components/schemas/coordinate"},
		},
	} {
		if err := g.RegisterParameter(name, p); err != nil {
			panic(err)
		}
	}

	badRequest := func(opts ...fizz.ResponseOption) fizz.OperationOption {
		return fizz.ResponseWith("400", "The provided parameters are incorrect",
			append([]fizz.ResponseOption{fizz.WithContent("text/html", errorMessage(""))}, opts...)...)
	}

	f.GET("/board", []fizz.OperationOption{
		fizz.ID("get-board"),
		fizz.Summary("Get the whole board"),
		fizz.Description("Retrieves the current state of the board and the winner."),
		fizz.Tags("Gameplay"),
		requirement("defaultApiKey"),
		requirement("app2AppOauth", "board:read"),
	}, tonic.Handler(func(*gin.Context) (*status, error) { return nil, nil }, 200))

	f.GET("/board/:row/:column", []fizz.OperationOption{
		fizz.ID("get-square"),
		fizz.Summary("Get a single board square"),
		fizz.Description("Retrieves the requested square."),
		fizz.Tags("Gameplay"),
		requirement("bearerHttpAuthentication"),
		requirement("user2AppOauth", "board:read"),
		badRequest(),
		fizz.CustomizeOperation(markSquareLink),
	}, tonic.Handler(func(*gin.Context, *squareIn) (mark, error) { return "", nil }, 200))

	f.PUT("/board/:row/:column", []fizz.OperationOption{
		fizz.ID("put-square"),
		fizz.Summary("Set a single board square"),
		fizz.Description("Places a mark on the board and retrieves the whole board and the winner (if any)."),
		fizz.Tags("Gameplay"),
		requirement("bearerHttpAuthentication"),
		requirement("user2AppOauth", "board:write"),
		fizz.RequestBody("", true, fizz.Content("application/json", mark(""))),
		fizz.Response("202", "Mark operation has not completed. Use callback to check for progress", nil,
			[]*openapi.ResponseHeader{{Name: "Location", Description: "Callback URL"}}, nil),
		badRequest(),
		fizz.CustomizeOperation(statusCallback),
	}, tonic.Handler(func(*gin.Context, *markIn) (*status, error) { return nil, nil }, 200))
	return f
}
