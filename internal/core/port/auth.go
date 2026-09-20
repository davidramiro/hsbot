package port

import "context"

type Authorizer interface {
	IsAuthorized(ctx context.Context, chatID int64) bool
}
