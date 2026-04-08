package ports

import "context"

type OTPGenerator interface {
	Generate(ctx context.Context) (string, error)
}
