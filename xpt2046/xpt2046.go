// xpt2046/xpt2046.go
package xpt2046

import (
	"machine"
)

// Point はタッチ座標を表す構造体
type Point struct {
	X, Y, Z int
}

type Device struct {
	bus      *machine.SPI
	cs       machine.Pin
	irq      machine.Pin
	rotation int
}

// New は新しいXPT2046デバイスを作成する
func New(bus *machine.SPI, cs, irq machine.Pin) *Device {
	cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs.High()

	// IRQピンは入力（プルアップなしでも動くことが多いが、必要なら設定）
	irq.Configure(machine.PinConfig{Mode: machine.PinInput})

	return &Device{
		bus: bus,
		cs:  cs,
		irq: irq,
	}
}

// ReadTouchPoint は現在のタッチ座標と圧力(Z)を読み取る
// Z > 0 ならタッチされていると判断できる
func (d *Device) ReadTouchPoint() Point {
	// 複数のサンプルを取って平均化すると安定するが、まずはシンプルに実装する

	// Z (圧力) の読み取り
	// コマンド: Start(1) | A2-A0(011=Z1) | Mode(0=12bit) | SER/DFR(0=Diff) | PD1-PD0(00=LowPower)
	z1 := d.readReg(0xB1)
	// コマンド: Start(1) | A2-A0(100=Z2) | ...
	z2 := d.readReg(0xC1)

	z := z1 + 4095 - z2
	if z < 0 {
		z = 0
	}

	// 圧力が低い（触っていない）場合は座標を読まずに帰る
	// 閾値は環境によるが、とりあえず小さめに設定
	if z < 400 {
		return Point{X: 0, Y: 0, Z: 0}
	}

	// X座標の読み取り (コマンド 0xD1: X position)
	x := d.readReg(0xD1)

	// Y座標の読み取り (コマンド 0x91: Y position)
	y := d.readReg(0x91)

	return Point{X: int(x), Y: int(y), Z: int(z)}
}

// readReg は指定したコマンドを送って12bitの値を読み取る
func (d *Device) readReg(cmd byte) uint16 {
	d.cs.Low()

	// コマンド送信
	d.bus.Transfer(cmd)

	// データ受信 (12bitなので2バイト読む)
	// 最初の1ビットはBusy、続く12ビットがデータ、残り3ビットは無視
	b1, _ := d.bus.Transfer(0x00)
	b2, _ := d.bus.Transfer(0x00)

	d.cs.High()

	// 12bitデータに変換
	// 受信データは [7ビット:データ上位] [5ビット:データ下位 + 3ビット:詰め物] のような形式で来る
	val := (uint16(b1) << 8) | uint16(b2)
	val = val >> 3 // 下位3ビットを捨てる

	return val
}
