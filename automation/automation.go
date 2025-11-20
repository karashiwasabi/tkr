// C:\Users\wasab\OneDrive\デスクトップ\TKR\automation\automation.go
package automation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/launcher"
)

// DownloadDat はMEDICODE-Webにログインし、DATファイルをダウンロードします。
func DownloadDat(userId, password, saveDir string) (string, error) {
	// 保存先ディレクトリの確保
	if _, err := os.Stat(saveDir); os.IsNotExist(err) {
		if err := os.MkdirAll(saveDir, 0755); err != nil {
			return "", fmt.Errorf("保存先フォルダの作成に失敗: %v", err)
		}
	}

	// 1. ブラウザ起動
	// Leakless(false) でセキュリティソフト対策
	u := launcher.New().
		Headless(false).
		Leakless(false).
		MustLaunch()

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	// 2. ログイン画面へ
	fmt.Println("MEDICODE-Webにアクセス中...")
	page := browser.MustPage("https://www.e-mednet.jp/")
	page.MustWaitStable()

	// 3. ログイン操作
	fmt.Println("ログイン情報を入力中...")

	if err := rod.Try(func() {
		page.MustElement("[name='userid']").MustInput(userId)
	}); err != nil {
		return "", fmt.Errorf("ユーザーID入力欄が見つかりません: %v", err)
	}

	if err := rod.Try(func() {
		page.MustElement("[name='userpsw']").MustInput(password)
	}); err != nil {
		return "", fmt.Errorf("パスワード入力欄が見つかりません: %v", err)
	}

	fmt.Println("ログインボタンをクリック...")
	loginBtn, err := page.ElementR("input, button, a, img", "ログイン")
	if err == nil {
		loginBtn.MustClick()
	} else {
		page.KeyActions().Press(input.Enter).MustDo()
	}

	page.MustWaitStable()

	// 4. メニュー移動
	fmt.Println("メニュー[納品処理]を検索中...")
	if err := rod.Try(func() {
		page.MustElementR("a, span, div, img", "納品処理").MustClick()
	}); err != nil {
		return "", fmt.Errorf("メニュー[納品処理]が見つかりません（ログイン失敗の可能性あり）: %v", err)
	}
	page.MustWaitStable()

	// 5. サブメニュー
	fmt.Println("メニュー[納品...受信...JAN]を検索中...")
	if err := rod.Try(func() {
		// URLの一部で検索 (最も確実な方法)
		page.MustElement("a[href*='SrDeliveryJanDownload']").MustClick()
	}); err != nil {
		return "", fmt.Errorf("メニュー[納品受信(JAN)]が見つかりません: %v", err)
	}
	page.MustWaitStable()

	// 6. ダウンロード準備
	wait := browser.MustWaitDownload()

	// ダイアログ（アラート）が出たら自動的にOKを押して閉じる設定
	go page.MustHandleDialog()

	// 7. ボタンクリック
	fmt.Println("ダウンロードボタンをクリック...")
	clicked := false
	selectors := []string{
		"input[value*='未受信データ']",
		"input[type='button']",
		"button",
	}

	for _, sel := range selectors {
		if el, err := page.ElementR(sel, "未受信"); err == nil {
			el.MustClick()
			clicked = true
			break
		}
	}

	if !clicked {
		return "", fmt.Errorf("「未受信データ全件受信」ボタンが見つかりませんでした")
	}

	// 8. 監視ループ (ダウンロード開始 vs 画面メッセージ変化)
	fmt.Println("ダウンロード待機中...")

	var fileData []byte
	resultChan := make(chan string)

	// A. ダウンロード監視
	go func() {
		// パニック対策
		defer func() {
			_ = recover()
		}()
		data := wait()
		fileData = data
		resultChan <- "downloaded"
	}()

	// B. 画面メッセージ監視
	go func() {
		// 最大30秒待つ
		for i := 0; i < 60; i++ {
			time.Sleep(500 * time.Millisecond)

			if body, err := page.Element("body"); err == nil {
				text, _ := body.Text()

				if strings.Contains(text, "ありませんでした") {
					resultChan <- "no_data"
					return
				}
			}
		}
	}()

	// どちらかが来るのを待つ
	select {
	case res := <-resultChan:
		if res == "no_data" {
			// 正常終了ステータスを返す
			return "NO_DATA", nil
		}
		// "downloaded" の場合は下に続く

	case <-time.After(60 * time.Second):
		return "", fmt.Errorf("処理がタイムアウトしました（ダウンロードもメッセージも確認できず）")
	}

	if len(fileData) == 0 {
		return "", fmt.Errorf("ダウンロードデータが空です")
	}

	// 9. ファイル保存
	fileName := fmt.Sprintf("MEDICODE_%s.DAT", time.Now().Format("20060102150405"))
	destPath := filepath.Join(saveDir, fileName)

	if err := os.WriteFile(destPath, fileData, 0644); err != nil {
		return "", fmt.Errorf("ファイルの書き込みに失敗: %v", err)
	}

	fmt.Printf("ダウンロード完了: %s\n", destPath)
	return destPath, nil
}
