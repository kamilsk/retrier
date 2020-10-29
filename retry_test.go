package retry_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	. "github.com/kamilsk/retry/v5"
	"github.com/kamilsk/retry/v5/strategy"
)

func TestDo(t *testing.T) {
	type expected struct {
		attempts uint
		error    error
	}

	tests := map[string]struct {
		breaker    strategy.Breaker
		strategies How
		action     func(context.Context) error
		expected   expected
	}{
		"success call": {
			breaker(),
			How{strategy.Wait(time.Hour)},
			func(context.Context) error { return nil },
			expected{1, nil},
		},
		"failure call": {
			breaker(),
			How{strategy.Limit(10)},
			func(context.Context) error { return layer{causer{errors.New("failure")}} },
			expected{10, layer{causer{errors.New("failure")}}},
		},
		"call with interrupted breaker": {
			interrupted(),
			How{strategy.Delay(time.Hour)},
			func(context.Context) error { return errors.New("zero iterations") },
			expected{0, context.Canceled},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var attempts uint
			action := func(ctx context.Context) error {
				attempts += 1
				return test.action(ctx)
			}
			err := Do(test.breaker, action, test.strategies...)
			if test.expected.attempts != attempts {
				t.Errorf("expected: %d, obtained: %d", test.expected.attempts, attempts)
			}
			if !reflect.DeepEqual(test.expected.error, err) {
				t.Error("result is not asserted")
			}
		})
	}

	t.Run("preserve context values", func(t *testing.T) {
		ctx := context.WithValue(context.TODO(), "test", "value")
		action := func(ctx context.Context) error {
			if !reflect.DeepEqual("value", ctx.Value("test")) {
				t.Error("value is not preserved")
			}
			return nil
		}
		if err := Do(ctx, action); err != nil {
			t.Error("result is not asserted")
		}
	})
}

func TestGo(t *testing.T) {
	type expected struct {
		attempts uint
		error    error
	}

	tests := map[string]struct {
		breaker    strategy.Breaker
		strategies How
		action     func(context.Context) error
		expected   expected
	}{
		"success call": {
			breaker(),
			How{strategy.Wait(time.Hour)},
			func(context.Context) error { return nil },
			expected{1, nil},
		},
		"failure call": {
			breaker(),
			How{strategy.Limit(10)},
			func(context.Context) error { return layer{causer{errors.New("failure")}} },
			expected{10, layer{causer{errors.New("failure")}}},
		},
		"call with interrupted breaker": {
			interrupted(),
			How{strategy.Delay(time.Hour)},
			func(context.Context) error { return errors.New("zero iterations") },
			expected{0, context.Canceled},
		},
		"call with panicked error": {
			breaker(),
			How{strategy.Wait(time.Hour)},
			func(context.Context) error { panic(errors.New("failure")) },
			expected{1, errors.New("failure")},
		},
		"call with non-error panic": {
			breaker(),
			How{strategy.Wait(time.Hour)},
			func(context.Context) error { panic("non-error") },
			expected{1, fmt.Errorf("retry: unexpected panic: %#v", "non-error")},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var attempts uint
			action := func(ctx context.Context) error {
				attempts += 1
				return test.action(ctx)
			}
			err := Go(test.breaker, action, test.strategies...)
			if test.expected.attempts != attempts {
				t.Errorf("expected: %d, obtained: %d", test.expected.attempts, attempts)
			}
			if !reflect.DeepEqual(test.expected.error, err) {
				t.Error("result is not asserted")
			}
		})
	}

	t.Run("preserve context values", func(t *testing.T) {
		ctx := context.WithValue(context.TODO(), "test", "value")
		action := func(ctx context.Context) error {
			if !reflect.DeepEqual("value", ctx.Value("test")) {
				t.Error("value is not preserved")
			}
			return nil
		}
		if err := Go(ctx, action); err != nil {
			t.Error("result is not asserted")
		}
	})
}

// helpers

func breaker() strategy.Breaker {
	return context.TODO()
}

func interrupted() strategy.Breaker {
	ctx, cancel := context.WithCancel(context.TODO())
	cancel()
	return ctx
}

type layer struct{ error }

func (layer layer) Unwrap() error { return layer.error }

type causer struct{ error }

func (causer causer) Cause() error { return causer.error }
