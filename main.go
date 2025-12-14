package main

// 配線情報については、プロジェクトルートの README.md を参照してください。

import (
	"fmt"
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/tinyfont/proggy"

	"testing-pico/cc1101"
	"testing-pico/display"
	"testing-pico/xpt2046"
)

func main() {
	// --- 1. SPIバスの設定 ---
	// SPI通信に使用するピンを設定
	machine.SPI0.Configure(machine.SPIConfig{
		Frequency: 4000000, // タッチパネルの仕様に合わせて4MHzに設定
		SCK:       machine.GP2,
		SDO:       machine.GP3, // MOSI
		SDI:       machine.GP4, // MISO (タッチパネル使用時に必須)
	})

	// 将来タッチパネルを使用するためのピン定義（予約）
	touchCS := machine.GP9   // タッチパネルのChip Select
	touchIRQ := machine.GP10 // タッチパネルの割り込み

	// --- 1-2. SPI1バスの設定 (CC1101用) ---
	// ディスプレイとは別のSPIバスを使用する
	machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 500000, // CC1101はそこまで高速でなくても良い
		SCK:       machine.GP14,
		SDO:       machine.GP15, // MOSI
		SDI:       machine.GP12, // MISO
	})

	// CC1101用のピン定義
	cc1101CS := machine.GP13
	cc1101GDO0 := machine.GP16
	cc1101GDO0.Configure(machine.PinConfig{Mode: machine.PinInput}) // 割り込み用に入力設定

	// --- 2. ディスプレイの初期化 ---
	// displayパッケージのInit関数を呼び出して初期化
	// 変数名はパッケージ名と重複しないよう lcd とする
	lcd := display.Init(machine.SPI0, machine.GP6, machine.GP5, machine.GP7, machine.GP8)

	// コンソール（ログ出力機能）の作成
	// 緑色の文字でログを出力する設定
	console := display.NewConsole(lcd, &proggy.TinySZ8pt7b, color.RGBA{R: 0, G: 255, B: 0, A: 255})

	console.Println("System Init...")
	console.Println("Display: OK")

	// タッチパネルの初期化
	touch := xpt2046.New(machine.SPI0, touchCS, touchIRQ)
	console.Println("Touch: Enabled")

	// --- 3. CC1101の初期化 ---
	console.Println("Init CC1101...")
	// GDO2は使用しないので NoPin を指定
	radio := cc1101.New(machine.SPI1, cc1101CS, cc1101GDO0, machine.NoPin)
	if !radio.Configure() {
		console.Error("CC1101: Init Failed!")
	} else {
		console.Println("CC1101: Ready (433MHz)")
		// 受信モード(RX)に切り替え、電波を待ち受ける
		radio.Rx()
	}

	counter := 0

	// --- 4. 無限ループ ---
	// プログラム終了を防ぐための無限ループ
	for {
		// 高速なループでパケットの取りこぼしを防ぐ
		time.Sleep(10 * time.Millisecond)
		counter++

		// 1. パケット受信チェック
		// GDO0がHighの場合、データ受信の可能性がある
		if cc1101GDO0.Get() {
			// データ読み出し用のバッファ（最大64バイト）
			data := make([]byte, 64)
			n, err := radio.Read(data)
			if err == nil && n > 0 {
				// 受信したデータを16進数と文字列で表示
				console.Println(fmt.Sprintf("RX[%d]: %X", n, data[:n]))
				console.Println(fmt.Sprintf("Text: %s", string(data[:n])))

				// 読み出し後はIDLE状態になることがあるので、再度受信モードにする
				radio.Rx()
			}
		}

		// 2. RSSI表示（頻繁すぎると見づらいので、10回に1回更新）
		if counter%10 == 0 {
			// CC1101からRSSIを取得
			rssi := radio.ReadRSSI()

			// 視覚的な表現としてバーグラフを生成
			barLen := (rssi + 100) / 5
			if barLen < 0 {
				barLen = 0
			}
			if barLen > 15 {
				barLen = 15
			}
			bar := ""
			for i := 0; i < barLen; i++ {
				bar += "|"
			}

			// 画面に表示
			msg := fmt.Sprintf("RSSI: %d %s", rssi, bar)
			console.Println(msg)
		}

		// 3. タッチパネルの入力チェック
		// 画面へのタッチを検出
		p := touch.ReadTouchPoint()
		if p.Z > 0 { // Z座標が0より大きい場合、タッチ（圧力）有りと判定
			// --- キャリブレーション設定 ---
			// 画面の端をタッチしてログに出力されるRaw X, Yの値を基に、この値を調整すること。
			// ※ 使用するモジュールの個体差に合わせて調整が必要である。
			const (
				rawMinX = 300
				rawMaxX = 3800
				rawMinY = 200
				rawMaxY = 3700
			)

			// --- 座標変換（マッピング） ---
			// Raw座標を画面座標(0-240, 0-320)に変換する
			// もし「右に動かしたのにカーソルが左に行く」などの逆転現象が起きる場合、
			// mapRangeの引数 `outMin` と `outMax` を入れ替えることで（例: 0, 240 -> 240, 0）、座標を反転できる。

			// まず X=X, Y=Y で対応付けを行う（もし軸が逆の場合は XとYを入れ替えること）。
			screenX := mapRange(p.X, rawMinX, rawMaxX, 0, 240)
			screenY := mapRange(p.Y, rawMinY, rawMaxY, 0, 320)

			// ログ出力（調整用にRaw値も見れるようにしておく）
			console.Println(fmt.Sprintf("Touch: Raw(%d,%d) -> Screen(%d,%d)", p.X, p.Y, screenX, screenY))

			// --- お絵かき機能 ---
			// タッチした座標に赤い点を描画する。
			lcd.FillRectangle(int16(screenX), int16(screenY), 2, 2, color.RGBA{255, 0, 0, 255})
		}
	}
}

// mapRange は、ある範囲の数値を別の範囲の数値に変換する関数である。
func mapRange(x, inMin, inMax, outMin, outMax int) int {
	val := (x-inMin)*(outMax-outMin)/(inMax-inMin) + outMin
	if val < outMin {
		return outMin
	}
	if val > outMax {
		return outMax
	}
	return val
}
