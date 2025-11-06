// C:\Users\wasab\OneDrive\デスクトップ\TKR\model\transaction_types.go
package model

type DatRecord struct {
	ClientCode    string
	Flag          int
	Date          string
	ReceiptNumber string
	LineNumber    string
	JanCode       string
	ProductName   string
	DatQuantity   float64
	UnitPrice     float64
	Subtotal      float64
	ExpiryDate    string
	LotNumber     string
}

type TransactionRecord struct {
	ID                  int     `db:"id" json:"id"`
	TransactionDate     string  `db:"transaction_date" json:"transactionDate"`
	ClientCode          string  `db:"client_code" json:"clientCode"`
	ReceiptNumber       string  `db:"receipt_number" json:"receiptNumber"`
	LineNumber          string  `db:"line_number" json:"lineNumber"`
	Flag                int     `db:"flag" json:"flag"`
	JanCode             string  `db:"jan_code" json:"janCode"`
	YjCode              string  `db:"yj_code" json:"yjCode"`
	ProductName         string  `db:"product_name" json:"productName"`
	KanaName            string  `db:"kana_name" json:"kanaName"`
	UsageClassification string  `db:"usage_classification" json:"usageClassification"`
	PackageForm         string  `db:"package_form" json:"packageForm"`
	PackageSpec         string  `db:"package_spec" json:"packageSpec"`
	MakerName           string  `db:"maker_name" json:"makerName"`
	DatQuantity         float64 `db:"dat_quantity" json:"datQuantity"`
	JanPackInnerQty     float64 `db:"jan_pack_inner_qty" json:"janPackInnerQty"`
	JanQuantity         float64 `db:"jan_quantity" json:"janQuantity"`
	JanPackUnitQty      float64 `db:"jan_pack_unit_qty" json:"janPackUnitQty"`
	JanUnitName         string  `db:"jan_unit_name" json:"janUnitName"`
	JanUnitCode         string  `db:"jan_unit_code" json:"janUnitCode"`
	YjQuantity          float64 `db:"yj_quantity" json:"yjQuantity"`
	YjPackUnitQty       float64 `db:"yj_pack_unit_qty" json:"yjPackUnitQty"`
	YjUnitName          string  `db:"yj_unit_name" json:"yjUnitName"`
	UnitPrice           float64 `db:"unit_price" json:"unitPrice"`
	PurchasePrice       float64 `db:"purchase_price" json:"purchasePrice"`
	SupplierWholesale   string  `db:"supplier_wholesale" json:"supplierWholesale"`
	Subtotal            float64 `db:"subtotal" json:"subtotal"`
	TaxAmount           float64 `db:"tax_amount" json:"taxAmount"`
	TaxRate             float64 `db:"tax_rate" json:"taxRate"`
	ExpiryDate          string  `db:"expiry_date" json:"expiryDate"`
	LotNumber           string  `db:"lot_number" json:"lotNumber"`
	FlagPoison          int     `db:"flag_poison" json:"flagPoison"`
	FlagDeleterious     int     `db:"flag_deleterious" json:"flagDeleterious"`
	FlagNarcotic        int     `db:"flag_narcotic" json:"flagNarcotic"`
	FlagPsychotropic    int     `db:"flag_psychotropic" json:"flagPsychotropic"`
	FlagStimulant       int     `db:"flag_stimulant" json:"flagStimulant"`
	FlagStimulantRaw    int     `db:"flag_stimulant_raw" json:"flagStimulantRaw"`
	ProcessFlagMA       string  `db:"process_flag_ma" json:"processFlagMA"`
}

type UnifiedInputRecord struct {
	Date        string
	YjCode      string
	JanCode     string
	ProductName string
	YjQuantity  float64
	YjUnitName  string
}
