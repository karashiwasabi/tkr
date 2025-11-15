// C:\Users\wasab\OneDrive\デスクトップ\TKR\model\order_types.go
package model

// Backorder は backorders テーブルのレコードを表します。
// (WASABI: model/types.go を TKR 用に修正)
type Backorder struct {
	ID        int    `db:"id" json:"id"`
	OrderDate string `db:"order_date" json:"orderDate"`
	// ▼▼▼【ここに追加】▼▼▼
	JanCode string `db:"jan_code" json:"janCode,omitempty"`
	// ▲▲▲【追加ここまで】▲▲▲
	YjCode            string  `db:"yj_code" json:"yjCode"`
	ProductName       string  `db:"product_name" json:"productName"`
	PackageForm       string  `db:"package_form" json:"packageForm"`
	JanPackInnerQty   float64 `db:"jan_pack_inner_qty" json:"janPackInnerQty"`
	YjUnitName        string  `db:"yj_unit_name" json:"yjUnitName"`
	OrderQuantity     float64 `db:"order_quantity" json:"orderQuantity"`
	RemainingQuantity float64 `db:"remaining_quantity" json:"remainingQuantity"`
	// TKRの database/backorders.go に合わせ、*sql.NullString ではなく string を使用
	WholesalerCode string  `db:"wholesaler_code" json:"wholesalerCode,omitempty"`
	YjPackUnitQty  float64 `db:"yj_pack_unit_qty" json:"yjPackUnitQty"`
	JanPackUnitQty float64 `db:"jan_pack_unit_qty" json:"janPackUnitQty"`
	JanUnitCode    int     `db:"jan_unit_code" json:"janUnitCode"`

	// フロントエンドからの発注データ受け取り用フィールド (DBカラムなし)
	YjQuantity float64 `json:"yjQuantity,omitempty"`
}

// ▼▼▼【ここから追加】 (WASABI: model/types.go  より) ▼▼▼
type PriceUpdate struct {
	ProductCode      string  `json:"productCode"`
	NewPurchasePrice float64 `json:"newPrice"`
	NewSupplier      string  `json:"newWholesaler"`
}

type ProductQuote struct {
	ProductCode    string  `db:"product_code" json:"productCode"`
	WholesalerCode string  `db:"wholesaler_code" json:"wholesalerCode"`
	QuotePrice     float64 `db:"quote_price" json:"quotePrice"`
	QuoteDate      string  `db:"quote_date" json:"quoteDate"`
}

// ▲▲▲【追加ここまで】▲▲▲
