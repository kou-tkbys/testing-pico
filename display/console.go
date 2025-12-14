package display

import (
	"image/color"

	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/tinyfont"
)

// Console は画面へのログ出力を管理する構造体
type Console struct {
	display *ili9341.Device
	font    *tinyfont.Font
	color   color.RGBA
	x, y    int16 // 現在のカーソル位置
	lineH   int16 // 1行の高さ
}

// NewConsole は新しいコンソールを作成する
func NewConsole(d *ili9341.Device, f *tinyfont.Font, c color.RGBA) *Console {
	return &Console{
		display: d,
		font:    f,
		color:   c,
		x:       10, // 左端のマージン
		y:       20, // 初期カーソルY位置
		lineH:   10, // 行の高さ
	}
}

// Println は通常のログ（基本色）を出力する
func (c *Console) Println(msg string) {
	c.log(msg, c.color)
}

// Warn は警告ログ（黄色）を出力する
func (c *Console) Warn(msg string) {
	c.log(msg, color.RGBA{255, 255, 0, 255})
}

// Error はエラーログ（赤色）を出力する
func (c *Console) Error(msg string) {
	c.log(msg, color.RGBA{255, 0, 0, 255})
}

// log は共通の描画処理（内部利用のみ）
func (c *Console) log(msg string, col color.RGBA) {
	// 画面サイズを取得
	_, h := c.display.Size()

	// 次の行が画面からはみ出るかチェック
	if c.y+c.lineH > h {
		// はみ出る場合は画面をクリアし、カーソルを上部に戻す
		c.display.FillScreen(color.RGBA{0, 0, 0, 255})
		c.y = 20 // 初期位置へリセット
	}

	// 文字列を描画
	tinyfont.WriteLine(c.display, c.font, c.x, c.y, msg, col)

	// カーソルを次の行へ移動
	c.y += c.lineH
}
