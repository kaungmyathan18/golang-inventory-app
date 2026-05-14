package handler

import (
	"errors"
	"net/http"

	"github.com/kaungmyathan18/golang-inventory-app/internal/apiresponse"
	"github.com/kaungmyathan18/golang-inventory-app/internal/repository"
)

func writeResourceError(w http.ResponseWriter, r *http.Request, err error, resource string) {
	if errors.Is(err, repository.ErrNotFound) {
		apiresponse.WriteProblem(w, r, http.StatusNotFound,
			apiresponse.ProblemTypeURI(r, "not-found"),
			"Not Found",
			"No "+resource+" exists for the given id.",
			nil,
		)
		return
	}
	apiresponse.WriteInternalError(w, r, err)
}

func writeMissingRelation(w http.ResponseWriter, r *http.Request, resource string) {
	apiresponse.WriteProblem(w, r, http.StatusUnprocessableEntity,
		apiresponse.ProblemTypeURI(r, "validation"),
		"Validation Failed",
		"The referenced "+resource+" does not exist.",
		nil,
	)
}
