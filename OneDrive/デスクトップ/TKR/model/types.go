// C:\Users\wasab\OneDrive\デスクトップ\TKR\model\types.go (全体)
package model

import "database/sql"

type JcshmsInfo struct {
	ProductCode string `db:"JC000" json:"productCode"`
	YjCode      string `db:"JC009" json:"yjCode"`
	ProductName string `db:"JC018" json:"productName"`

	// ▼▼▼【ここに追加】▼▼▼
	Specification string `db:"JC020" json:"specification"` // 規格容量
	// ▲▲▲【追加ここまで】▲▲▲

	KanaNameShort string `db:"JC019" json:"kanaNameShort"`

	KanaName string `db:"JC022" json:"kanaName"`

	GenericName string `db:"JC024" json:"genericName"`

	MakerName           string  `db:"JC030" json:"makerName"`
	UsageClassification string  `db:"JC013" json:"usageClassification"`
	PackageForm         string  `db:"JC037" json:"packageForm"`
	YjUnitName          string  `db:"JC039" json:"yjUnitName"`
	YjPackUnitQty       float64 `db:"JC044" json:"yjPackUnitQty"`
	NhiPrice            float64 `db:"JC049" json:"nhiPrice"`
	PackageNhiPrice     float64 `db:"JC050" json:"packageNhiPrice"`
	Gs1Code             string  `db:"JC122" json:"gs1Code"`
	NhiPriceFactor      float64 `db:"JC124" json:"nhiPriceFactor"`

	JanPackInnerQty sql.NullFloat64 `db:"JA006" json:"janPackInnerQty"`
	JanUnitCode     sql.NullString  `db:"JA007" json:"janUnitCode"`
	JanPackUnitQty  sql.NullFloat64 `db:"JA008" json:"janPackUnitQty"`

	FlagPoison       int `db:"JC061" json:"flagPoison"`
	FlagDeleterious  int `db:"JC062" json:"flagDeleterious"`
	FlagNarcotic     int `db:"JC063" json:"flagNarcotic"`
	FlagPsychotropic int `db:"JC064" json:"flagPsychotropic"`
	FlagStimulant    int `db:"JC065" json:"flagStimulant"`
	FlagStimulantRaw int `db:"JC066" json:"flagStimulantRaw"`
}

type ProductMaster struct {
	ProductCode string `db:"product_code" json:"productCode"`
	YjCode      string `db:"yj_code" json:"yjCode"`
	Gs1Code     string `db:"gs1_code" json:"gs1Code"`
	ProductName string `db:"product_name" json:"productName"`
	KanaName    string `db:"kana_name" json:"kanaName"`

	KanaNameShort string `db:"kana_name_short" json:"kanaNameShort"`
	GenericName   string `db:"generic_name" json:"genericName"`

	MakerName           string  `db:"maker_name" json:"makerName"`
	Specification       string  `db:"specification" json:"specification"`
	UsageClassification string  `db:"usage_classification" json:"usageClassification"`
	PackageForm         string  `db:"package_form" json:"packageForm"`
	YjUnitName          string  `db:"yj_unit_name" json:"yjUnitName"`
	YjPackUnitQty       float64 `db:"yj_pack_unit_qty" json:"yjPackUnitQty"`
	JanPackInnerQty     float64 `db:"jan_pack_inner_qty" json:"janPackInnerQty"`
	JanUnitCode         int     `db:"jan_unit_code" json:"janUnitCode"`
	JanPackUnitQty      float64 `db:"jan_pack_unit_qty" json:"janPackUnitQty"`
	Origin              string  `db:"origin" json:"origin"`
	NhiPrice            float64 `db:"nhi_price" json:"nhiPrice"`
	PurchasePrice       float64 `db:"purchase_price" json:"purchasePrice"`
	FlagPoison          int     `db:"flag_poison" json:"flagPoison"`
	FlagDeleterious     int     `db:"flag_deleterious" json:"flagDeleterious"`
	FlagNarcotic        int     `db:"flag_narcotic" json:"flagNarcotic"`
	FlagPsychotropic    int     `db:"flag_psychotropic" json:"flagPsychotropic"`
	FlagStimulant       int     `db:"flag_stimulant" json:"flagStimulant"`
	FlagStimulantRaw    int     `db:"flag_stimulant_raw" json:"flagStimulantRaw"`
	IsOrderStopped      int     `db:"is_order_stopped" json:"isOrderStopped"`
	SupplierWholesale   string  `db:"supplier_wholesale" json:"supplierWholesale"`
	GroupCode           string  `db:"group_code" json:"groupCode"`
	ShelfNumber         string  `db:"shelf_number" json:"shelfNumber"`
	Category            string  `db:"category" json:"category"`
	UserNotes           string  `db:"user_notes" json:"userNotes"`
}

type ProductMasterInput struct {
	ProductCode string `db:"product_code" json:"productCode"`
	YjCode      string `db:"yj_code" json:"yjCode"`
	Gs1Code     string `db:"gs1_code" json:"gs1Code"`
	ProductName string `db:"product_name" json:"productName"`
	KanaName    string `db:"kana_name" json:"kanaName"`

	KanaNameShort string `db:"kana_name_short" json:"kanaNameShort"`
	GenericName   string `db:"generic_name" json:"genericName"`

	MakerName           string  `db:"maker_name" json:"makerName"`
	Specification       string  `db:"specification" json:"specification"`
	UsageClassification string  `db:"usage_classification" json:"usageClassification"`
	PackageForm         string  `db:"package_form" json:"packageForm"`
	YjUnitName          string  `db:"yj_unit_name" json:"yjUnitName"`
	YjPackUnitQty       float64 `db:"yj_pack_unit_qty" json:"yjPackUnitQty"`
	JanPackInnerQty     float64 `db:"jan_pack_inner_qty" json:"janPackInnerQty"`
	JanUnitCode         int     `db:"jan_unit_code" json:"janUnitCode"`
	JanPackUnitQty      float64 `db:"jan_pack_unit_qty" json:"janPackUnitQty"`
	Origin              string  `db:"origin" json:"origin"`
	NhiPrice            float64 `db:"nhi_price" json:"nhiPrice"`
	PurchasePrice       float64 `db:"purchase_price" json:"purchasePrice"`
	FlagPoison          int     `db:"flag_poison" json:"flagPoison"`
	FlagDeleterious     int     `db:"flag_deleterious" json:"flagDeleterious"`
	FlagNarcotic        int     `db:"flag_narcotic" json:"flagNarcotic"`
	FlagPsychotropic    int     `db:"flag_psychotropic" json:"flagPsychotropic"`
	FlagStimulant       int     `db:"flag_stimulant" json:"flagStimulant"`
	FlagStimulantRaw    int     `db:"flag_stimulant_raw" json:"flagStimulantRaw"`
	IsOrderStopped      int     `db:"is_order_stopped" json:"isOrderStopped"`
	SupplierWholesale   string  `db:"supplier_wholesale" json:"supplierWholesale"`
	GroupCode           string  `db:"group_code" json:"groupCode"`
	ShelfNumber         string  `db:"shelf_number" json:"shelfNumber"`
	Category            string  `db:"category" json:"category"`
	UserNotes           string  `db:"user_notes" json:"userNotes"`
}

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

// ▼▼▼【ここから追加】未定義だった型を追加 ▼▼▼

type Client struct {
	ClientCode string `db:"client_code" json:"clientCode"`
	ClientName string `db:"client_name" json:"clientName"`
}

type Wholesaler struct {
	WholesalerCode string `db:"wholesaler_code" json:"wholesalerCode"`
	WholesalerName string `db:"wholesaler_name" json:"wholesalerName"`
}

// usage_parser.go で使用
type UnifiedInputRecord struct {
	Date        string
	YjCode      string
	JanCode     string
	ProductName string
	YjQuantity  float64
	YjUnitName  string
}

// ▲▲▲【追加ここまで】▲▲▲
