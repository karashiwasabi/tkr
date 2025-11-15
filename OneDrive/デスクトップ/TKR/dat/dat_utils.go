// C:\Users\wasab\OneDrive\デスクトップ\TKR\dat\dat_utils.go
package dat

import (
	"encoding/json"
	"log"
	"net/http"
)

// respondJSONError は dat パッケージ共有のエラーレスポンス関数です
func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Println("Error response:", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": message,
		"results": []interface{}{},
	})
}
