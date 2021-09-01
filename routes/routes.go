package routes

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/jwtauth"
	"github.com/go-chi/render"
	"github.com/go-resty/resty/v2"
	"github.com/kr/pretty"
)

// ErrResponse renderer type for handling all sorts of errors.
//
// In the best case scenario, the excellent github.com/pkg/errors package
// helps reveal information on the error, setting it on Err, and in the Render()
// method, using it to set the application-specific error code in AppCode.
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

type SignupRequest struct {
	// User *User `json:"user,omitempty"`
	*User
	// ProtectedID string `json:"id"` // override 'id' json to have more control
}

type User struct {
	ID       string
	Email    string
	Password string
}

type SignupResponse struct {
	*User
	Token   string
	Elapsed int
}

func NewUserCreatedResponse(user *User, token string) *SignupResponse {
	resp := &SignupResponse{
		User:  user,
		Token: token,
	}

	return resp
}

func dbNewUser(user *User) {
	// Create a resty object
	client := resty.New()
	// query the Hasura query endpoint
	// to create a new user
	gqlEndpoint := os.Getenv("GRAPHQL_ENDPOINT")
	// queryName := "CreateUserQuery"

	// insert a user with email and password
	// using the graphql API
	body := fmt.Sprintf(`
		{
			"operationName": "CreateUserMutation",
			"query":
			"mutation MyQuery {
				insert_users_one(
							  object: {
								  "name": "Kaushik Varanasi",
								  "email": %s,
								  "passwordhash": %s,
							  }
				) {
				  id
				}
			}"
		}
	`, user.Email, user.Password)
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(gqlEndpoint)

	pretty.Println(resp, err)
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func (rd *SignupResponse) Render(w http.ResponseWriter, r *http.Request) error {
	// Pre-processing before a response is marshalled and sent across the wire
	rd.Elapsed = 10
	return nil
}

func (u *SignupRequest) Bind(r *http.Request) error {
	// u.User is nil if no User fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if u.User == nil {
		return errors.New("missing required Article fields.")
	}
	// a.User is nil if no Userpayload fields are sent in the request. In this app
	// this won't cause a panic, but checks in this Bind method may be required if
	// a.User or futher nested fields like a.User.Name are accessed elsewhere.

	// just a post-process after a decode..
	u.User.ID = "" // unset the protected ID
	// a.Article.Title = strings.ToLower(a.Article.Title) // as an example, we down-case
	return nil
}

func ChiSignupHandler(w http.ResponseWriter, r *http.Request) {
	data := &SignupRequest{}

	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	user := &User{
		Email:    data.Email,
		Password: data.Password,
	}
	tokenAuth := jwtauth.New("HS256", []byte("secret"), nil)

	// For debugging/example purposes, we generate and print
	// a sample jwt token with claims `user_id:123` here:
	_, tokenString, _ := tokenAuth.Encode(map[string]interface{}{"user_id": data.Email})
	fmt.Printf("DEBUG: a sample jwt is %s\n\n", tokenString)
	token := tokenString
	dbNewUser(user)
	// pretty.Println(users)

	render.Status(r, http.StatusCreated)
	render.Render(w, r, NewUserCreatedResponse(user, token))
}
