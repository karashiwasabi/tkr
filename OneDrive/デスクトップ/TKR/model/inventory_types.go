// C:\Users\wasab\OneDrive\デスクトップ\TKR\model\inventory_types.go
package model

// DeadStockRecord はロット・期限・在庫（JAN単位）の記録です。
type DeadStockRecord struct {
	// ▼▼▼【ここに追加】(WASABI: model/types.go [cite: 1025-1026] より) ▼▼▼
	ID int `db:"id" json:"id"`
	// ▲▲▲【追加ここまで】▲▲▲
	ProductCode      string  `db:"product_code" json:"productCode"`
	YjCode           string  `db:"yj_code" json:"yjCode"`
	PackageForm      string  `db:"package_form" json:"packageForm"`
	JanPackInnerQty  float64 `db:"jan_pack_inner_qty" json:"janPackInnerQty"`
	YjUnitName       string  `db:"yj_unit_name" json:"yjUnitName"`
	StockQuantityJan float64 `db:"stock_quantity_jan" json:"stockQuantityJan"`
	ExpiryDate       string  `db:"expiry_date" json:"expiryDate"`
	LotNumber        string  `db:"lot_number" json:"lotNumber"`
}

// PackageStock は package_stock テーブルのレコードです。
type PackageStock struct {
	PackageKey        string  `db:"package_key"`
	YjCode            string  `db:"yj_code"`
	StockQuantityYj   float64 `db:"stock_quantity_yj"`
	LastInventoryDate string  `db:"last_inventory_date"`
}

// DeadStockItem は不動在庫リストの表示用構造体です。
type DeadStockItem struct {
	PackageKey      string  `db:"package_key" json:"packageKey"`
	YjCode          string  `db:"yj_code" json:"yjCode"`
	StockQuantityYj float64 `db:"stock_quantity_yj" json:"stockQuantityYj"`
	// ▼▼▼【ここに追加】JAN数量（計算結果）と内包装数量（計算用）▼▼▼
	StockQuantityJan float64 `json:"stockQuantityJan"` // JSONに追加
	JanPackInnerQty  float64 `db:"jan_pack_inner_qty"` // DBから取得
	// ▲▲▲【追加ここまで】▲▲▲
	ProductName         string      `db:"product_name" json:"productName"` // 代表品名
	PackageSpec         string      `db:"package_spec" json:"packageSpec"` // 代表包装仕様
	LotDetails          []LotDetail `json:"lotDetails"`                    // 棚卸明細
	KanaName            string      `db:"kana_name"`
	UsageClassification string      `db:"usage_classification"`
}

// LotDetail は棚卸時のロット・期限・JAN数量の明細です。
type LotDetail struct {
	JanCode     string  `db:"jan_code" json:"JanCode"`
	Gs1Code     string  `db:"gs1_code" json:"Gs1Code"`
	PackageSpec string  `db:"package_spec" json:"PackageSpec"`
	ExpiryDate  string  `db:"expiry_date" json:"ExpiryDate"`
	LotNumber   string  `db:"lot_number" json:"LotNumber"`
	JanQuantity float64 `db:"jan_quantity" json:"JanQuantity"`
	JanUnitName string  `db:"jan_unit_name" json:"JanUnitName"`
}
