// C:\Users\wasab\OneDrive\デスクトップ\TKR\precomp\handler.go
package precomp

import (
	"encoding/json"
	"log"
	"net/http"
	"tkr/database" // TKRのdatabaseを参照
	"tkr/model"    // TKRのmodelを参照

	"github.com/jmoiron/sqlx" // TKRはsqlxを使用
)

// PrecompPayload は保存・更新時にフロントエンドから受け取るデータ構造です
// (WASABI: precomp/handler.go [cite: 1475] より)
type PrecompPayload struct {
	PatientNumber string                        `json:"patientNumber"`
	Records       []database.PrecompRecordInput `json:"records"`
}

// LoadResponse は呼び出し時にフロントエンドへ返すデータ構造です
// (WASABI: precomp/handler.go [cite: 1476] より)
type LoadResponse struct {
	Status  string                    `json:"status"`
	Records []model.TransactionRecord `json:"records"`
}

// SavePrecompHandler は予製データを保存・更新します
// (WASABI: precomp/handler.go [cite: 1477-1478] より。 *sql.DB -> *sqlx.DB)
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

		tx, err := conn.Beginx() // sqlx.Tx
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// TKRの DBTX (sqlx.Tx が満たす) を渡す
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

// LoadPrecompHandler は患者の予製データと現在のステータスを返します
// (WASABI: precomp/handler.go [cite: 1478-1479] より。 *sql.DB -> *sqlx.DB)
func LoadPrecompHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientNumber := r.URL.Query().Get("patientNumber")
		if patientNumber == "" {
			http.Error(w, "Patient number is required", http.StatusBadRequest)
			return
		}

		// TKRの DBTX (sqlx.DB が満たす) を渡す
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

// ClearPrecompHandler は予製データを完全に削除します
// (WASABI: precomp/handler.go [cite: 1480-1481] より。 *sql.DB -> *sqlx.DB)
func ClearPrecompHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientNumber := r.URL.Query().Get("patientNumber")
		if patientNumber == "" {
			http.Error(w, "Patient number is required", http.StatusBadRequest)
			return
		}

		// TKRの DBTX (sqlx.DB が満たす) を渡す
		if err := database.DeletePreCompoundingRecordsByPatient(conn, patientNumber); err != nil {
			http.Error(w, "Failed to clear pre-compounding records: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "予製データを完全に削除しました。"})
	}
}

// SuspendPrecompHandler は予製を中断状態にします
// (WASABI: precomp/handler.go [cite: 1481-1482] より。 *sql.DB -> *sqlx.DB)
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
		tx, err := conn.Beginx() // sqlx.Tx
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// TKRの DBTX (sqlx.Tx が満たす) を渡す
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

// ResumePrecompHandler は予製を再開状態にします
// (WASABI: precomp/handler.go [cite: 1482-1483] より。 *sql.DB -> *sqlx.DB)
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
		tx, err := conn.Beginx() // sqlx.Tx
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// TKRの DBTX (sqlx.Tx が満たす) を渡す
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

// GetStatusPrecompHandler は予製の現在の状態を返します
// (WASABI: precomp/handler.go [cite: 1483-1484] より。 *sql.DB -> *sqlx.DB)
func GetStatusPrecompHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		patientNumber := r.URL.Query().Get("patientNumber")
		if patientNumber == "" {
			http.Error(w, "Patient number is required", http.StatusBadRequest)
			return
		}

		// TKRの DBTX (sqlx.DB が満たす) を渡す
		status, err := database.GetPreCompoundingStatusByPatient(conn, patientNumber)
		if err != nil {
			http.Error(w, "Failed to get status: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": status})
	}
}

// (WASABI: precomp/handler.go [cite: 1484-1488] の Export/Import ハンドラは、TKRへの移植指示には含まれていなかったため、一旦スキップします。)
