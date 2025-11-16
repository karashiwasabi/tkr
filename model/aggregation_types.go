// C:\Users\wasab\OneDrive\デスクトップ\TKR\model\aggregation_types.go
package model

// AggregationFilters は集計時のフィルタ条件です。
type AggregationFilters struct {
	StartDate    string
	EndDate      string
	KanaName     string
	DrugTypes    []string
	DosageForm   string
	Coefficient  float64
	YjCode       string
	MovementOnly bool
	ShelfNumber  string
	GenericName  string
}

// StockLedgerYJGroup はYJコード単位の集計グループです。
type StockLedgerYJGroup struct {
	YjCode                string                    `json:"yjCode"`
	ProductName           string                    `json:"productName"`
	YjUnitName            string                    `json:"yjUnitName"`
	PackageLedgers        []StockLedgerPackageGroup `json:"packageLedgers"`
	StartingBalance       interface{}               `json:"startingBalance"`
	NetChange             float64                   `json:"netChange"`
	EndingBalance         interface{}               `json:"endingBalance"`
	TotalReorderPoint     float64                   `json:"totalReorderPoint"`
	IsReorderNeeded       bool                      `json:"isReorderNeeded"`
	TotalBaseReorderPoint float64                   `json:"totalBaseReorderPoint"`
	TotalPrecompounded    float64                   `json:"totalPrecompounded"`
}

// StockLedgerPackageGroup は包装キー単位の集計グループです。
type StockLedgerPackageGroup struct {
	PackageKey             string              `json:"packageKey"`
	JanUnitName            string              `json:"janUnitName"`
	StartingBalance        interface{}         `json:"startingBalance"`
	Transactions           []LedgerTransaction `json:"transactions"`
	NetChange              float64             `json:"netChange"`
	EndingBalance          interface{}         `json:"endingBalance"`
	Masters                []*ProductMaster    `json:"masters"`
	EffectiveEndingBalance float64             `json:"effectiveEndingBalance"`
	MaxUsage               float64             `json:"maxUsage"`
	ReorderPoint           float64             `json:"reorderPoint"`
	IsReorderNeeded        bool                `json:"isReorderNeeded"`
	BaseReorderPoint       float64             `json:"baseReorderPoint"`
	PrecompoundedTotal     float64             `json:"precompoundedTotal"`
}

// LedgerTransaction は台帳表示用の取引記録です。
type LedgerTransaction struct {
	TransactionRecord
	RunningBalance float64 `json:"runningBalance"`
}

// ValuationFilters は在庫評価の絞り込み条件です。
type ValuationFilters struct {
	Date                string
	KanaName            string
	UsageClassification string // (JSからは dosageForm として渡される)
}

// ValuationDetailRow は在庫評価の明細行データです。
type ValuationDetailRow struct {
	YjCode               string  `json:"yjCode"`
	ProductName          string  `json:"productName"`
	ProductCode          string  `json:"productCode"` // 代表JAN
	PackageSpec          string  `json:"packageSpec"`
	Stock                float64 `json:"stock"` // YJ単位
	YjUnitName           string  `json:"yjUnitName"`
	PackageNhiPrice      float64 `json:"packageNhiPrice"`
	PackagePurchasePrice float64 `json:"packagePurchasePrice"`
	TotalNhiValue        float64 `json:"totalNhiValue"`
	TotalPurchaseValue   float64 `json:"totalPurchaseValue"`
	ShowAlert            bool    `json:"showAlert"`
	// ▼▼▼【ここから追加】CSV出力用に内包装数量を追加 ▼▼▼
	PackageKey      string  `json:"packageKey"`
	JanPackInnerQty float64 `json:"janPackInnerQty"`
	// ▲▲▲【追加ここまで】▲▲▲
}
