package forms

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"reflect"

	"github.com/go-playground/mold/v4"
	"github.com/go-playground/mold/v4/modifiers"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/schema"
	"github.com/microcosm-cc/bluemonday"
)

type FormProcessor interface {
	Process(dst any, req *http.Request) error
}

type formProcessorImpl struct {
	decoder  *schema.Decoder
	validate *validator.Validate
	policy   *bluemonday.Policy
	modifier *mold.Transformer
}

func NewFormProcessor() (FormProcessor, error) {
	decoder := schema.NewDecoder()
	validate := validator.New(validator.WithRequiredStructEnabled())
	policy := bluemonday.StrictPolicy()
	modifier := modifiers.New()

	return NewFormProcessorInitialized(
		decoder,
		validate,
		policy,
		modifier,
	)
}

func NewFormProcessorInitialized(
	decoder *schema.Decoder,
	validate *validator.Validate,
	policy *bluemonday.Policy,
	modifier *mold.Transformer,
) (FormProcessor, error) {
	if decoder == nil || validate == nil || policy == nil {
		return nil, errors.New("FormProcessor dependencies not fulfilled")
	}

	modifier.Register("sanitize", func(ctx context.Context, fl mold.FieldLevel) error {
		switch fl.Field().Kind() {
		case reflect.String:
			fl.Field().SetString(policy.Sanitize(fl.Field().String()))
		}
		return nil
	})

	return &formProcessorImpl{
		decoder:  decoder,
		validate: validate,
		policy:   policy,
		modifier: modifier,
	}, nil
}

var (
	ErrorParsing    = errors.New("error parsing form")
	ErrorDecoding   = errors.New("error decoding form")
	ErrorModifying  = errors.New("error modifying form")
	ErrorValidating = errors.New("error validating form")
)

// parses request form and mutates dst
func (fp *formProcessorImpl) Process(dst interface{}, req *http.Request) error {
	if err := req.ParseForm(); err != nil {
		slog.Error("ProcessForm", "step", "parse", "err", err)
		return ErrorParsing
	}
	if err := fp.decoder.Decode(dst, req.PostForm); err != nil {
		slog.Error("ProcessForm", "step", "decode", "err", err)
		return ErrorDecoding
	}
	if err := fp.modifier.Struct(context.Background(), dst); err != nil {
		slog.Error("ProcessForm", "step", "modify", "err", err)
		return ErrorModifying
	}
	if err := fp.validate.Struct(dst); err != nil {
		slog.Error("ProcessForm", "step", "validate", "err", err)
		return ErrorValidating
	}

	return nil
}
