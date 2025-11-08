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
	YjCode          string                    `json:"yjCode"`
	ProductName     string                    `json:"productName"`
	YjUnitName      string                    `json:"yjUnitName"`
	PackageLedgers  []StockLedgerPackageGroup `json:"packageLedgers"`
	StartingBalance interface{}               `json:"startingBalance"`
	NetChange       float64                   `json:"netChange"`
	EndingBalance   interface{}               `json:"endingBalance"`
	// ▼▼▼【ここから追加】(WASABI: model/types.go  より) ▼▼▼
	TotalReorderPoint     float64 `json:"totalReorderPoint"`
	IsReorderNeeded       bool    `json:"isReorderNeeded"`
	TotalBaseReorderPoint float64 `json:"totalBaseReorderPoint"`
	TotalPrecompounded    float64 `json:"totalPrecompounded"`
	// ▲▲▲【追加ここまで】▲▲▲
}

// StockLedgerPackageGroup は包装キー単位の集計グループです。
type StockLedgerPackageGroup struct {
	PackageKey      string              `json:"packageKey"`
	JanUnitName     string              `json:"janUnitName"`
	StartingBalance interface{}         `json:"startingBalance"`
	Transactions    []LedgerTransaction `json:"transactions"`
	NetChange       float64             `json:"netChange"`
	EndingBalance   interface{}         `json:"endingBalance"`
	Masters         []*ProductMaster    `json:"masters"`
	// ▼▼▼【ここから追加】(WASABI: model/types.go  より) ▼▼▼
	EffectiveEndingBalance float64 `json:"effectiveEndingBalance"`
	MaxUsage               float64 `json:"maxUsage"`
	ReorderPoint           float64 `json:"reorderPoint"`
	IsReorderNeeded        bool    `json:"isReorderNeeded"`
	BaseReorderPoint       float64 `json:"baseReorderPoint"`
	PrecompoundedTotal     float64 `json:"precompoundedTotal"`
	// ▲▲▲【追加ここまで】▲▲▲
}

// LedgerTransaction は台帳表示用の取引記録です。
type LedgerTransaction struct {
	TransactionRecord
	RunningBalance float64 `json:"runningBalance"`
}
