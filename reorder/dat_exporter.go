// C:\Users\wasab\OneDrive\デスクトップ\TKR\reorder\dat_exporter.go
package reorder

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// DatOrderRequest はフロントエンドから送られてくる発注データの構造です
type DatOrderRequest struct {
	JanCode        string  `json:"janCode"`
	WholesalerCode string  `json:"wholesalerCode"`
	OrderQuantity  float64 `json:"orderQuantity"`
	KanaNameShort  string  `json:"kanaNameShort"` // 商品名
}

// DatBuilder は固定長DATデータを作成するための構造体です
type DatBuilder struct {
	pharmacyID string // 薬局ID (MedicodeUserID)
	buffer     *bytes.Buffer
}

func NewDatBuilder(pharmacyID string) *DatBuilder {
	return &DatBuilder{
		pharmacyID: pharmacyID,
		buffer:     new(bytes.Buffer),
	}
}

// convertToSJIS は文字列をShift_JISバイト列に変換します
func convertToSJIS(s string) ([]byte, error) {
	encoder := japanese.ShiftJIS.NewEncoder()
	encoded, _, err := transform.Bytes(encoder, []byte(s))
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// formatField は指定された文字列をShift_JISバイト列に変換し、
// 指定されたバイト長(byteLen)に整形（右スペース埋め/切り詰め）して返します。
func formatField(text string, byteLen int) []byte {
	// 1. そのままShift_JISバイト列へ変換
	sjisBytes, err := convertToSJIS(text)
	if err != nil {
		// 変換エラー時は安全策として空スペースで埋める
		return bytes.Repeat([]byte(" "), byteLen)
	}

	// 2. 長さ調整 (バイト数基準)
	if len(sjisBytes) > byteLen {
		// 指定長より長い場合は切り詰める
		return sjisBytes[:byteLen]
	} else if len(sjisBytes) < byteLen {
		// 指定長より短い場合は右側をスペース(0x20)で埋める
		padding := bytes.Repeat([]byte(" "), byteLen-len(sjisBytes))
		return append(sjisBytes, padding...)
	}

	// ちょうど良い長さ
	return sjisBytes
}

// S行 (ヘッダー) をバッファに書き込みます
// 構成: S10(3) + 薬局ID(10) + 卸コード(9) + 予備(1) + 伝票番号(4) + 日時(12) + "128128"(6) + 予備(83) = 128
func (b *DatBuilder) writeSRecord(wholesalerCode string, slipNumber int) {
	// 現在日時 (YYMMDDHHMMSS)
	now := time.Now().Format("060102150405")

	// 各フィールドを作成
	var record []byte
	record = append(record, []byte("S10")...)                                   // 識別 (3)
	record = append(record, formatField(b.pharmacyID, 10)...)                   // 薬局ID (10)
	record = append(record, formatField(wholesalerCode, 9)...)                  // 卸コード (9)
	record = append(record, []byte(" ")...)                                     // 予備 (1)
	record = append(record, formatField(fmt.Sprintf("%04d", slipNumber), 4)...) // 伝票番号 (4)
	record = append(record, formatField(now, 12)...)                            // 日時 (12)
	record = append(record, []byte("128128")...)                                // ブロック長 (6)

	// ここまでで 3+10+9+1+4+12+6 = 45バイト
	// 残り 128 - 45 = 83バイトをスペースで埋める
	paddingLen := 128 - len(record)
	if paddingLen > 0 {
		record = append(record, bytes.Repeat([]byte(" "), paddingLen)...)
	}

	b.buffer.Write(record)
}

// D行 (明細) をバッファに書き込みます
// 構成: D10(3) + 薬局ID(10) + 納入日(6) + 予備(10) + 商品コード(14) + 商品名(40) + 数量(10) + 予備(10) + 不明(8) + 予備(17) = 128
func (b *DatBuilder) writeDRecord(item DatOrderRequest) {
	today := time.Now().Format("060102")

	// JANコード変換: 13桁なら先頭に '1' を付与して14桁にする
	jan := item.JanCode
	if len(jan) == 13 {
		jan = "1" + jan
	}

	// 数量変換: 整数5桁 + 小数5桁 (例: 1 -> 0000100000)
	qtyVal := int(item.OrderQuantity * 100000)
	qtyStr := fmt.Sprintf("%010d", qtyVal)

	// 各フィールドを作成
	var record []byte
	record = append(record, []byte("D10")...)                       // 識別 (3)
	record = append(record, formatField(b.pharmacyID, 10)...)       // 薬局ID (10)
	record = append(record, formatField(today, 6)...)               // 納入日 (6)
	record = append(record, bytes.Repeat([]byte(" "), 10)...)       // 予備 (10)
	record = append(record, formatField(jan, 14)...)                // 商品コード (14)
	record = append(record, formatField(item.KanaNameShort, 40)...) // 商品名 (40)
	record = append(record, formatField(qtyStr, 10)...)             // 数量 (10)

	// ★ご指摘のスペース (10桁)
	record = append(record, bytes.Repeat([]byte(" "), 10)...) // 予備 (10)

	record = append(record, []byte("00000000")...) // 不明 (8)

	// ここまでで 3+10+6+10+14+40+10+10+8 = 111バイト
	// 残り 128 - 111 = 17バイトをスペースで埋める
	paddingLen := 128 - len(record)
	if paddingLen > 0 {
		record = append(record, bytes.Repeat([]byte(" "), paddingLen)...)
	} else if paddingLen < 0 {
		// 万が一オーバーしていたら128バイトで切る
		record = record[:128]
	}

	b.buffer.Write(record)
}

// E行 (トレーラー) をバッファに書き込みます
// 構成: E10(3) + 総行数(6) + 明細数(6) + 予備(113) = 128
func (b *DatBuilder) writeERecord(totalLines, itemLines int) {
	// 総行数、明細数を6桁ゼロ埋め
	totalStr := fmt.Sprintf("%06d", totalLines)
	itemStr := fmt.Sprintf("%06d", itemLines)

	var record []byte
	record = append(record, []byte("E10")...)    // 識別 (3)
	record = append(record, []byte(totalStr)...) // 総行数 (6)
	record = append(record, []byte(itemStr)...)  // 明細数 (6)

	// ここまでで 3+6+6 = 15バイト
	// 残り 128 - 15 = 113バイトをスペースで埋める
	paddingLen := 128 - len(record)
	if paddingLen > 0 {
		record = append(record, bytes.Repeat([]byte(" "), paddingLen)...)
	}

	b.buffer.Write(record)
}

// GenerateFixedLengthDat は発注リクエストのリストからDATファイル(バイト列)を生成します。
func GenerateFixedLengthDat(pharmacyID string, requests []DatOrderRequest) ([]byte, error) {
	if pharmacyID == "" {
		return nil, fmt.Errorf("薬局ID(MedicodeUserID)が設定されていません")
	}

	builder := NewDatBuilder(pharmacyID)

	// 卸ごとにグルーピング
	ordersByWholesaler := make(map[string][]DatOrderRequest)
	for _, req := range requests {
		if req.WholesalerCode == "" {
			continue
		}
		ordersByWholesaler[req.WholesalerCode] = append(ordersByWholesaler[req.WholesalerCode], req)
	}

	// 卸コード順にソートして処理
	var wholesalers []string
	for w := range ordersByWholesaler {
		wholesalers = append(wholesalers, w)
	}
	sort.Strings(wholesalers)

	slipCounter := 1 // 伝票番号 (連番)

	for _, wCode := range wholesalers {
		items := ordersByWholesaler[wCode]
		if len(items) == 0 {
			continue
		}

		// S行
		builder.writeSRecord(wCode, slipCounter)

		// D行
		for _, item := range items {
			builder.writeDRecord(item)
		}

		// E行
		// 総行数 = S(1) + D(len) + E(1)
		totalLines := 1 + len(items) + 1
		builder.writeERecord(totalLines, len(items))

		slipCounter++
	}

	return builder.buffer.Bytes(), nil
}
