package main

import (
	"errors"
	"net/http"

	"github.com/igredk/greenlight/internal/data"
	"github.com/igredk/greenlight/internal/validator"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	// Create an anonymous struct to hold the expected data from the request body.
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	// Parse the request body into the anonymous struct.
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	// Copy the data from the request body into a new User struct.
	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		// Manually add a message to the validator instance, and then call failedValidationResponse() helper.
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Launch a background goroutine which runs an anonymous function that sends the welcome email.
	app.background(func() {
		err = app.mailer.Send(user.Email, "user_welcome.html", user)
		if err != nil {
			// Use the app.logger.PrintError() helper instead of the app.serverErrorResponse() helper
			// to prevent writing a second HTTP response and getting
			// "http: superfluous response.WriteHeader call" error from http.Server.
			app.logger.PrintError(err, nil)
		}
	})

	// Write a JSON response containing the user data along with a 201 Created status code.
	err = app.writeJSON(w, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
