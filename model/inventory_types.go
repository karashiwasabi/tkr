package model

type DeadStockRecord struct {
	ID               int     `db:"id" json:"id"`
	ProductCode      string  `db:"product_code" json:"productCode"`
	YjCode           string  `db:"yj_code" json:"yjCode"`
	PackageForm      string  `db:"package_form" json:"packageForm"`
	JanPackInnerQty  float64 `db:"jan_pack_inner_qty" json:"janPackInnerQty"`
	YjUnitName       string  `db:"yj_unit_name" json:"yjUnitName"`
	StockQuantityJan float64 `db:"stock_quantity_jan" json:"stockQuantityJan"`
	ExpiryDate       string  `db:"expiry_date" json:"expiryDate"`
	LotNumber        string  `db:"lot_number" json:"lotNumber"`
}

type PackageStock struct {
	PackageKey        string  `db:"package_key"`
	YjCode            string  `db:"yj_code"`
	StockQuantityYj   float64 `db:"stock_quantity_yj"`
	LastInventoryDate string  `db:"last_inventory_date"`
}

type DeadStockItem struct {
	PackageKey          string      `db:"package_key" json:"packageKey"`
	YjCode              string      `db:"yj_code" json:"yjCode"`
	StockQuantityYj     float64     `db:"stock_quantity_yj" json:"stockQuantityYj"`
	CurrentStockYj      float64     `json:"currentStockYj"`
	StockQuantityJan    float64     `json:"stockQuantityJan"`
	JanPackInnerQty     float64     `db:"jan_pack_inner_qty" json:"janPackInnerQty"`
	ProductName         string      `db:"product_name" json:"productName"`
	PackageSpec         string      `db:"package_spec" json:"packageSpec"`
	LotDetails          []LotDetail `json:"lotDetails"`
	KanaName            string      `db:"kana_name"`
	UsageClassification string      `db:"usage_classification"`
}

type LotDetail struct {
	JanCode     string  `db:"jan_code" json:"JanCode"`
	Gs1Code     string  `db:"gs1_code" json:"Gs1Code"`
	PackageSpec string  `db:"package_spec" json:"PackageSpec"`
	ExpiryDate  string  `db:"expiry_date" json:"ExpiryDate"`
	LotNumber   string  `db:"lot_number" json:"LotNumber"`
	JanQuantity float64 `db:"jan_quantity" json:"JanQuantity"`
	JanUnitName string  `db:"jan_unit_name" json:"JanUnitName"`
}
