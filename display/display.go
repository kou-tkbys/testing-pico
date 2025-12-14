package display

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers/ili9341"
)

// Init はディスプレイとバックライトを初期化し、使用可能な状態にする
func Init(spi *machine.SPI, dc, cs, rst, bl machine.Pin) *ili9341.Device {
	// ディスプレイの初期化
	d := ili9341.NewSPI(
		spi,
		dc,
		cs,
		rst,
	)

	// バックライトの設定
	bl.Configure(machine.PinConfig{Mode: machine.PinOutput})
	bl.High()

	// ディスプレイの向き設定
	d.Configure(ili9341.Config{
		Rotation: ili9341.Rotation0, // 縦向き設定
	})

	// 画面を黒でクリアする
	d.FillScreen(color.RGBA{R: 0, G: 0, B: 0, A: 255})

	return d
}
