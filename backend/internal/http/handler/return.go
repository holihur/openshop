package handler

import "github.com/holihur/openshop/internal/domain"

// ReturnView is the API shape of a return request (RMA).
type ReturnView struct {
	ID        string `json:"id"`
	OrderID   string `json:"orderId"`
	UserID    string `json:"userId"`
	Reason    string `json:"reason"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

func ToReturnView(r domain.ReturnRequest) ReturnView {
	return ReturnView{
		ID: r.ID, OrderID: r.OrderID, UserID: r.UserID, Reason: r.Reason,
		Status: string(r.Status), CreatedAt: r.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}
