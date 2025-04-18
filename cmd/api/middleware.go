package main

import (
	"GoTodo/internal/data"
	"GoTodo/internal/data/validator"
	"errors"
	"net/http"
	"strings"
)

func (app *application) protectedRouteMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			app.invalidAuthenticationHeaderResponse(w, r)
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationHeaderResponse(w, r)
			return
		}

		token := headerParts[1]

		v := validator.New()

		if data.ValidateTokenPlainText(v, token); !v.Valid() {
			app.invalidAuthenticationHeaderResponse(w, r)
			return
		}

		user, err := app.models.Tokens.GetForToken(token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationHeaderResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}

			return
		}

		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}
