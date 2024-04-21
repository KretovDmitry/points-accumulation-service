package reward

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/KretovDmitry/gophermart-loyalty-service/internal/config"
	"github.com/KretovDmitry/gophermart-loyalty-service/internal/models/errs"
	"github.com/KretovDmitry/gophermart-loyalty-service/internal/models/order"
	"github.com/KretovDmitry/gophermart-loyalty-service/internal/models/user"
	"github.com/KretovDmitry/gophermart-loyalty-service/pkg/logger"
)

type Service struct {
	repo   Repository
	logger logger.Logger
	config *config.Config
}

func NewService(repo Repository, logger logger.Logger, config *config.Config) (*Service, error) {
	if config == nil {
		return nil, errors.New("nil dependency: config")
	}
	return &Service{repo: repo, logger: logger, config: config}, nil
}

var _ ServerInterface = (*Service)(nil)

func (s *Service) CreateOrder(w http.ResponseWriter, r *http.Request, params PostOrderParams) {
	u, found := user.FromContext(r.Context())
	if !found {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	order := &order.Order{
		UserID: u.ID,
		Number: params.Number,
		Status: order.NEW,
	}

	if err := s.repo.CreateOrder(r.Context(), order); err != nil {
		if !errors.Is(err, errs.ErrDataConflict) {
			ErrorHandlerFunc(w, r, err)
			return
		}
	}

	order, err := s.repo.GetOrderByNumber(r.Context(), params.Number)
	if err != nil {
		return
	}

	if order.UserID != u.ID {
		w.WriteHeader(http.StatusConflict)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ErrorHandlerFunc handles sending of an error in the JSON format,
// writing appropriate status code and handling the failure to marshal that.
func ErrorHandlerFunc(w http.ResponseWriter, _ *http.Request, err error) {
	errJSON := errs.JSON{Error: err.Error()}
	code := http.StatusInternalServerError

	switch {
	// Status Bad Request.
	case errors.Is(err, errs.ErrRequiredBodyParam) ||
		errors.Is(err, errs.ErrInvalidPayload) ||
		errors.Is(err, errs.ErrInvalidContentType):
		code = http.StatusBadRequest

	// Status Unauthorized.
	case errors.Is(err, errs.ErrNotFound) ||
		errors.Is(err, errs.ErrInvalidCredentials):
		code = http.StatusUnauthorized

	// Status Conflict.
	case errors.Is(err, errs.ErrDataConflict):
		code = http.StatusConflict
	}

	w.WriteHeader(code)

	if err = json.NewEncoder(w).Encode(errJSON); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}