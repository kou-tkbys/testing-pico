// cc1101/cc1101.go
package cc1101

import (
	"machine"
	"time"
)

// レジスタ定義 (必要なものだけ抜粋)
const (
	IOCFG2   = 0x00
	IOCFG0   = 0x02
	FIFOTHR  = 0x03
	PKTLEN   = 0x06
	PKTCTRL1 = 0x07
	PKTCTRL0 = 0x08
	ADDR     = 0x09
	CHANNR   = 0x0A
	FSCTRL1  = 0x0B
	FSCTRL0  = 0x0C
	FREQ2    = 0x0D
	FREQ1    = 0x0E
	FREQ0    = 0x0F
	MDMCFG4  = 0x10
	MDMCFG3  = 0x11
	MDMCFG2  = 0x12
	MDMCFG1  = 0x13
	MDMCFG0  = 0x14
	DEVIATN  = 0x15
	MCSM0    = 0x18
	FOCCFG   = 0x19
	BSCFG    = 0x1A
	AGCCTRL2 = 0x1B
	AGCCTRL1 = 0x1C
	AGCCTRL0 = 0x1D
	FREND1   = 0x21
	FREND0   = 0x22
	FSCAL3   = 0x23
	FSCAL2   = 0x24
	FSCAL1   = 0x25
	FSCAL0   = 0x26
	TEST0    = 0x2C
	PATABLE  = 0x3E
	FIFO     = 0x3F
)

// ストローブコマンド
const (
	SRES    = 0x30 // リセット
	SFSTXON = 0x31
	SXOFF   = 0x32
	SCAL    = 0x33
	SRX     = 0x34 // 受信モード
	STX     = 0x35 // 送信モード
	SIDLE   = 0x36 // アイドルモード
	SFRX    = 0x3A // RX FIFOフラッシュ
	SFTX    = 0x3B // TX FIFOフラッシュ
	SNOP    = 0x3D
)

// ステータスレジスタ (読み取り専用)
const (
	RSSI      = 0x34
	MARCSTATE = 0xF5
)

type Device struct {
	bus  *machine.SPI
	cs   machine.Pin
	gdo0 machine.Pin
	gdo2 machine.Pin
}

// New は新しいCC1101デバイスを作成する
func New(bus *machine.SPI, cs, gdo0, gdo2 machine.Pin) *Device {
	cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs.High()
	return &Device{
		bus:  bus,
		cs:   cs,
		gdo0: gdo0,
		gdo2: gdo2,
	}
}

// Configure はCC1101を初期化し、433MHz設定を適用する
func (d *Device) Configure() bool {
	// リセット
	d.strobe(SRES)
	time.Sleep(10 * time.Millisecond)

	// 基本設定 (433.92MHz, GFSK, 1.2kbps)
	// SmartRF Studioからの推奨値をベースにしている
	d.writeReg(IOCFG2, 0x29)   // GDO2: Chip RDY
	d.writeReg(IOCFG0, 0x06)   // GDO0: Sync Word Received
	d.writeReg(FIFOTHR, 0x47)  // FIFO Threshold
	d.writeReg(PKTCTRL1, 0x04) // Append Status, No Address Check
	d.writeReg(PKTCTRL0, 0x05) // Variable Packet Length, CRC Enabled
	d.writeReg(ADDR, 0x00)     // Device Address
	d.writeReg(CHANNR, 0x00)   // Channel 0

	// 周波数設定 433.92MHz
	d.writeReg(FSCTRL1, 0x06)
	d.writeReg(FSCTRL0, 0x00)
	d.writeReg(FREQ2, 0x10)
	d.writeReg(FREQ1, 0xB1)
	d.writeReg(FREQ0, 0x3B)

	// モデム設定 (GFSK, 1.2kbps)
	d.writeReg(MDMCFG4, 0xF5)
	d.writeReg(MDMCFG3, 0x83)
	d.writeReg(MDMCFG2, 0x13) // GFSK, DC Filter, Sync Mode
	d.writeReg(MDMCFG1, 0x22)
	d.writeReg(MDMCFG0, 0xF8)
	d.writeReg(DEVIATN, 0x15)

	d.writeReg(MCSM0, 0x18) // Auto Calibrate
	d.writeReg(FOCCFG, 0x16)
	d.writeReg(BSCFG, 0x6C)

	d.writeReg(AGCCTRL2, 0x03)
	d.writeReg(AGCCTRL1, 0x40)
	d.writeReg(AGCCTRL0, 0x91)

	d.writeReg(FREND1, 0x56)
	d.writeReg(FREND0, 0x10)

	d.writeReg(FSCAL3, 0xE9)
	d.writeReg(FSCAL2, 0x2A)
	d.writeReg(FSCAL1, 0x00)
	d.writeReg(FSCAL0, 0x1F)

	d.writeReg(TEST0, 0x59)

	// パワー設定 (10dBm)
	d.writeBurst(PATABLE, []byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	return true
}

// Rx は受信モードに移行する
func (d *Device) Rx() {
	d.strobe(SRX)
}

// Tx はデータを送信する
func (d *Device) Tx(packet []byte) {
	d.strobe(SIDLE)
	d.strobe(SFTX) // Flush TX FIFO

	// FIFOにデータを書き込む (Length + Data)
	d.writeReg(FIFO, byte(len(packet)))
	d.writeBurst(FIFO, packet)

	d.strobe(STX) // 送信開始

	// 送信完了まで待つ（GDO0が下がるのを待つなどの処理が必要だが簡易的にWait）
	time.Sleep(100 * time.Millisecond)
}

// ReadRSSI は現在のRSSI（信号強度）をdBmで返す
func (d *Device) ReadRSSI() int {
	rssiDec := int(d.readStatus(RSSI))
	if rssiDec >= 128 {
		return (rssiDec-256)/2 - 74
	}
	return rssiDec/2 - 74
}

// Read は受信したデータを読み出す
func (d *Device) Read(buf []byte) (int, error) {
	// RX FIFOにあるバイト数を確認
	// (PKTCTRL1でAppend Statusが有効な場合、最後の2バイトはステータス)
	// ここでは簡易的に実装

	// 1バイト読み出して長さ取得
	pktLen := int(d.readReg(FIFO))
	if pktLen > 64 || pktLen == 0 {
		d.strobe(SFRX) // 変なデータならフラッシュ
		return 0, nil
	}

	// バッファサイズに合わせて読み込む長さを決定
	readLen := pktLen
	if len(buf) < readLen {
		readLen = len(buf)
	}

	// データ読み出し
	for i := 0; i < readLen; i++ {
		buf[i] = d.readReg(FIFO)
	}

	// バッファに入りきらなかった残りのデータがあれば空読みして捨てる
	for i := readLen; i < pktLen; i++ {
		d.readReg(FIFO)
	}

	// ステータスバイトの読み出し (2バイト)
	// 1バイト目: RSSI, 2バイト目: LQI (ビット7がCRC_OK)
	_ = d.readReg(FIFO)
	lqi := d.readReg(FIFO)

	// CRCチェック (LQIのビット7が1ならOK)
	if lqi&0x80 == 0 {
		// CRCエラー（ノイズや衝突）の場合は無視する
		return 0, nil
	}

	return readLen, nil
}

// --- 内部ヘルパー関数 ---

func (d *Device) strobe(cmd byte) {
	d.cs.Low()
	d.bus.Transfer(cmd)
	d.cs.High()
}

func (d *Device) readReg(addr byte) byte {
	d.cs.Low()
	d.bus.Transfer(addr | 0x80) // Read bit set
	val, _ := d.bus.Transfer(0x00)
	d.cs.High()
	return val
}

func (d *Device) readStatus(addr byte) byte {
	d.cs.Low()
	d.bus.Transfer(addr | 0xC0) // Read & Burst bit (for status)
	val, _ := d.bus.Transfer(0x00)
	d.cs.High()
	return val
}

func (d *Device) writeReg(addr, val byte) {
	d.cs.Low()
	d.bus.Transfer(addr)
	d.bus.Transfer(val)
	d.cs.High()
}

func (d *Device) writeBurst(addr byte, data []byte) {
	d.cs.Low()
	d.bus.Transfer(addr | 0x40) // Burst bit set
	for _, b := range data {
		d.bus.Transfer(b)
	}
	d.cs.High()
}
