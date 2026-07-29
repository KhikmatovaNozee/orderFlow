package order

import "github.com/KhikmatovaNozee/orderFlow/internal/model"

func CanTransition(from, to string) bool {
	switch from {
	case model.OrderStatusNew:
		return to == model.OrderStatusPaid || to == model.OrderStatusShipped

	case model.OrderStatusPaid:
		return to == model.OrderStatusShipped

	default:
		return false
	}
}
