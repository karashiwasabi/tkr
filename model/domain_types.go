// C:\Users\wasab\OneDrive\デスクトップ\TKR\model\domain_types.go
package model

type Client struct {
	ClientCode string `db:"client_code" json:"clientCode"`
	ClientName string `db:"client_name" json:"clientName"`
}

type Wholesaler struct {
	WholesalerCode string `db:"wholesaler_code" json:"wholesalerCode"`
	WholesalerName string `db:"wholesaler_name" json:"wholesalerName"`
}
