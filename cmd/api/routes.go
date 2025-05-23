package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/todos", app.protectedRouteMiddleware(app.createTodoHandler))
	router.HandlerFunc(http.MethodGet, "/v1/todos", app.protectedRouteMiddleware(app.listTodosHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/todos/:id", app.protectedRouteMiddleware(app.deleteTodoHandler))
	router.HandlerFunc(http.MethodPut, "/v1/todos/:id", app.protectedRouteMiddleware(app.updateTodoHandler))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.createUserHandler)

	router.HandlerFunc(http.MethodPost, "/v1/auth/sign-in", app.createAuthenticationTokenHandler)

	return router
}
