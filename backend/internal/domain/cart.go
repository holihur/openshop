package domain

import "time"

// CartItem is a denormalised snapshot kept in the cart so the UI can render
// without hitting the catalog on every request. Totals are recomputed from the
// catalog at checkout time to avoid stale pricing.
type CartItem struct {
	ProductID  string    `json:"productId"`
	Title      string    `json:"title"`
	CoverImage string    `json:"coverImage"`
	PriceCents int64     `json:"priceCents"`
	Currency   string    `json:"currency"`
	Quantity   int       `json:"quantity"`
	AddedAt    time.Time `json:"addedAt"`
}

type Cart struct {
	UserID    string     `json:"userId"`
	Items     []CartItem `json:"items"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

func (c *Cart) TotalCents() int64 {
	var total int64
	for _, it := range c.Items {
		total += it.PriceCents * int64(it.Quantity)
	}
	return total
}

func (c *Cart) TotalQuantity() int {
	var q int
	for _, it := range c.Items {
		q += it.Quantity
	}
	return q
}

func (c *Cart) find(productID string) int {
	for i, it := range c.Items {
		if it.ProductID == productID {
			return i
		}
	}
	return -1
}

// AddItem inserts or increments a line. Quantity is clamped to at least one.
func (c *Cart) AddItem(item CartItem) {
	if item.Quantity < 1 {
		item.Quantity = 1
	}
	if i := c.find(item.ProductID); i >= 0 {
		c.Items[i].Quantity += item.Quantity
		// Refresh the snapshot with the latest catalog data.
		c.Items[i].Title = item.Title
		c.Items[i].CoverImage = item.CoverImage
		c.Items[i].PriceCents = item.PriceCents
		c.Items[i].Currency = item.Currency
		return
	}
	item.AddedAt = time.Now().UTC()
	c.Items = append(c.Items, item)
}

func (c *Cart) SetQuantity(productID string, quantity int) {
	i := c.find(productID)
	if i < 0 {
		return
	}
	if quantity <= 0 {
		c.RemoveItem(productID)
		return
	}
	c.Items[i].Quantity = quantity
}

func (c *Cart) RemoveItem(productID string) {
	i := c.find(productID)
	if i < 0 {
		return
	}
	c.Items = append(c.Items[:i], c.Items[i+1:]...)
}

func (c *Cart) Clear() { c.Items = nil }
