package models

type ClientOrder struct {
	ClientID string `gorm:"primaryKey" json:"client_id"`
	SortOrder int    `gorm:"default:0" json:"sort_order"`
}

func (ClientOrder) TableName() string {
	return "client_orders"
}
