// C:\Users\wasab\OneDrive\デスクトップ\TKR\model\inventory_types.go
package model

// DeadStockRecord はロット・期限・在庫（JAN単位）の記録です。
type DeadStockRecord struct {
	ID               int     `json:"id"`
	ProductCode      string  `json:"productCode"`
	YjCode           string  `json:"yjCode"`
	PackageForm      string  `json:"packageForm"`
	JanPackInnerQty  float64 `json:"janPackInnerQty"`
	YjUnitName       string  `json:"yjUnitName"`
	StockQuantityJan float64 `json:"stockQuantityJan"`
	ExpiryDate       string  `json:"expiryDate"`
	LotNumber        string  `json:"lotNumber"`
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
	PackageKey          string      `db:"package_key" json:"packageKey"`
	YjCode              string      `db:"yj_code" json:"yjCode"`
	StockQuantityYj     float64     `db:"stock_quantity_yj" json:"stockQuantityYj"`
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
