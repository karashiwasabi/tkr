package precomp

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
	"tkr/database"
	"tkr/model"
	"tkr/parsers"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

type PrecompPayload struct {
	PatientNumber string                        `json:"patientNumber"`
	Records       []database.PrecompRecordInput `json:"records"`
}

type LoadResponse struct {
	Status  string                    `json:"status"`
	Records []model.TransactionRecord `json:"records"`
}

func SavePrecompHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload PrecompPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if payload.PatientNumber == "" {
			http.Error(w, "Patient number is required", http.StatusBadRequest)
			return
		}

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := database.UpsertPreCompoundingRecordsInTx(tx, payload.PatientNumber, payload.Records); err != nil {
			log.Printf("ERROR: Failed to save pre-compounding records for patient %s: %v", payload.PatientNumber, err)
			http.Error(w, "Failed to save pre-compounding records: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "予製データを保存しました。"})
	}
}

func LoadPrecompHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientNumber := r.URL.Query().Get("patientNumber")
		if patientNumber == "" {
			http.Error(w, "Patient number is required", http.StatusBadRequest)
			return
		}

		status, err := database.GetPreCompoundingStatusByPatient(conn, patientNumber)
		if err != nil {
			http.Error(w, "Failed to get pre-compounding status: "+err.Error(), http.StatusInternalServerError)
			return
		}
		records, err := database.GetPreCompoundingRecordsByPatient(conn, patientNumber)
		if err != nil {
			http.Error(w, "Failed to load pre-compounding records: "+err.Error(), http.StatusInternalServerError)
			return
		}

		response := LoadResponse{
			Status:  status,
			Records: records,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func ClearPrecompHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientNumber := r.URL.Query().Get("patientNumber")
		if patientNumber == "" {
			http.Error(w, "Patient number is required", http.StatusBadRequest)
			return
		}

		if err := database.DeletePreCompoundingRecordsByPatient(conn, patientNumber); err != nil {
			http.Error(w, "Failed to clear pre-compounding records: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "予製データを完全に削除しました。"})
	}
}

func SuspendPrecompHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			PatientNumber string `json:"patientNumber"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if payload.PatientNumber == "" {
			http.Error(w, "Patient number is required", http.StatusBadRequest)
			return
		}
		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := database.SuspendPreCompoundingRecordsByPatient(tx, payload.PatientNumber); err != nil {
			http.Error(w, "Failed to suspend pre-compounding records: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "予製を中断しました。"})
	}
}

func ResumePrecompHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			PatientNumber string `json:"patientNumber"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if payload.PatientNumber == "" {
			http.Error(w, "Patient number is required", http.StatusBadRequest)
			return
		}
		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := database.ResumePreCompoundingRecordsByPatient(tx, payload.PatientNumber); err != nil {
			http.Error(w, "Failed to resume pre-compounding records: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "予製を再開しました。"})
	}
}

func GetStatusPrecompHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientNumber := r.URL.Query().Get("patientNumber")
		if patientNumber == "" {
			http.Error(w, "Patient number is required", http.StatusBadRequest)
			return
		}

		status, err := database.GetPreCompoundingStatusByPatient(conn, patientNumber)
		if err != nil {
			http.Error(w, "Failed to get status: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": status})
	}
}

func ExportAllPrecompHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		records, err := database.GetAllPreCompoundingRecords(db)
		if err != nil {
			http.Error(w, "Failed to get all precomp records: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		buf.Write([]byte{0xEF, 0xBB, 0xBF})
		writer := csv.NewWriter(&buf)

		header := []string{
			"patient_number", "product_code", "product_name", "quantity_jan", "unit_name",
		}
		if err := writer.Write(header); err != nil {
			http.Error(w, "Failed to write CSV header: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for _, rec := range records {
			row := []string{
				rec.ClientCode,
				rec.JanCode,
				rec.ProductName,
				fmt.Sprintf("%.2f", rec.JanQuantity),
				units.ResolveName(rec.JanUnitName),
			}
			if err := writer.Write(row); err != nil {
				log.Printf("WARN: Failed to write precomp row to CSV (Patient: %s): %v", rec.ClientCode, err)
			}
		}
		writer.Flush()

		if err := writer.Error(); err != nil {
			http.Error(w, "Failed to flush CSV writer: "+err.Error(), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("TKR予製データ(全件)_%s.csv", time.Now().Format("20060102"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))
		w.Write(buf.Bytes())
	}
}

func ImportAllPrecompHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "CSVファイルの読み取りに失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// ▼▼▼【ここを修正】戻り値の型を変更 ▼▼▼
		records, err := parsers.ParsePrecompCSV(file)
		if err != nil {
			// ▲▲▲【修正ここまで】▲▲▲
			http.Error(w, "CSVファイルの解析に失敗: "+err.Error(), http.StatusBadRequest)
			return
		}

		if len(records) == 0 {
			http.Error(w, "CSVから読み込むデータがありません。", http.StatusBadRequest)
			return
		}

		// ▼▼▼【ここから修正】新しい `records` スライス（ParsedPrecompCSVRecord）からマップを構築 ▼▼▼
		recordsByPatient := make(map[string][]database.PrecompRecordInput)
		for _, rec := range records {
			patientNumber := rec.PatientNumber
			recordsByPatient[patientNumber] = append(recordsByPatient[patientNumber], database.PrecompRecordInput{
				ProductCode: rec.ProductCode,
				JanQuantity: rec.JanQuantity,
			})
		}
		// ▲▲▲【修正ここまで】▲▲▲

		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "データベーストランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// ▼▼▼【ここから修正】`patientMap` ではなく `recordsByPatient` のキーをイテレート ▼▼▼
		processedPatients := 0
		for patientNumber, patientRecords := range recordsByPatient {

			if err := database.UpsertPreCompoundingRecordsInTx(tx, patientNumber, patientRecords); err != nil {
				http.Error(w, fmt.Sprintf("患者 %s の予製データ登録に失敗: %v", patientNumber, err), http.StatusInternalServerError)
				return
			}
			processedPatients++
		}
		// ▲▲▲【修正ここまで】▲▲▲

		if err := tx.Commit(); err != nil {
			http.Error(w, "データベースのコミットに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("%d名の患者の予製データをインポート（洗い替え）しました。", processedPatients),
		})
	}
}
