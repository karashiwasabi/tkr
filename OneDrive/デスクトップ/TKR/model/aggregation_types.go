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
	IsReorderNeeded bool                      `json:"isReorderNeeded"`
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
}

// LedgerTransaction は台帳表示用の取引記録です。
type LedgerTransaction struct {
	TransactionRecord
	RunningBalance float64 `json:"runningBalance"`
}
