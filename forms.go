package forms

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
	"github.com/microcosm-cc/bluemonday"
)

type FormProcessor struct {
	decoder  *schema.Decoder
	validate *validator.Validate
	policy   *bluemonday.Policy
}

func NewFormProcessor() (*FormProcessor, error) {
	decoder := schema.NewDecoder()
	validate := validator.New(validator.WithRequiredStructEnabled())
	policy := bluemonday.StrictPolicy()

	return NewFormProcessorInitialized(
		decoder,
		validate,
		policy,
	)
}

func NewFormProcessorInitialized(
	decoder *schema.Decoder,
	validate *validator.Validate,
	policy *bluemonday.Policy,
) (*FormProcessor, error) {
	if decoder == nil || validate == nil || policy == nil {
		return nil, errors.New("FormProcessor dependencies not fulfilled")
	}

	return &FormProcessor{
		decoder:  decoder,
		validate: validate,
		policy:   policy,
	}, nil
}

var (
	ErrorParsing    = errors.New("Error parsing form")
	ErrorDecoding   = errors.New("Error decoding form")
	ErrorValidating = errors.New("Error validating form")
)

// parses request form and mutates dst
func (fp *FormProcessor) ProcessForm(dst interface{}, req *http.Request) error {
	if err := req.ParseForm(); err != nil {
		slog.Error("ProcessForm", "step", "parse", "err", err)
		return ErrorParsing
	}
	if err := fp.decoder.Decode(dst, req.PostForm); err != nil {
		slog.Error("ProcessForm", "step", "decode", "err", err)
		return ErrorDecoding
	}
	if err := fp.validate.Struct(dst); err != nil {
		slog.Error("ProcessForm", "step", "validate", "err", err)
		return ErrorValidating
	}

	slog.Debug("ProcessForm", "step", "done")
	return nil
}
