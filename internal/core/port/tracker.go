package port

import "context"

type Tracker interface {
	AddCost(chatID int64, cost float64)
	CheckLimit(ctx context.Context, chatID int64) bool
	GetSpent(chatID int64) float64
}
