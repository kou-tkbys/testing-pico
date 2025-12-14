# Raspberry Pi Pico - RF & Touch Display Terminal

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Raspberry Pi PicoとTinyGoで実現する、CC1101無線モジュールとILI9341タッチディスプレイを組み合わせた多機能テストコンソール。

<p align="center"><img src="./docs/device_movie.gif" width="400"></p>

---

## 概要 (Overview)

このプロジェクトは、Raspberry Pi Pico（またはPico 2）を使い、ディスプレイへのログ出力、タッチパネルによる入力、そしてCC1101無線モジュールでのデータ受信を同時に行うためのテストプラットフォームです。

PCとのシリアル接続なしに、デバイス単体で無線データの受信状況やRSSI（受信強度）、タッチ座標などをリアルタイムに確認できます。

## 主な機能 (Features)

-   **ディスプレイコンソール**:
    -   ILI9341ディスプレイ（240x320）にログをリアルタイムで表示。
    -   テキストの自動折り返し、色分け表示に対応。
-   **RF受信**:
    -   CC1101モジュール（433MHz）で受信したデータを画面に表示。
    -   現在のRSSI（受信信号強度）をバーグラフで可視化。
-   **タッチパネル**:
    -   XPT2046タッチコントローラーからの入力を検知。
    -   タッチした座標に点を描画するお絵かき機能。
    -   タッチ座標のキャリブレーション（調整）のための情報出力。

## 必要なハードウェア (Hardware)

| コンポーネント      | 詳細                                               | 役割                             |
| ------------------- | -------------------------------------------------- | -------------------------------- |
| **MCU**             | Raspberry Pi Pico または Pico 2                    | メインコントローラー               |
| **Display & Touch** | ILI9341搭載 TFT LCD (2.4"など) + XPT2046タッチパネル | 表示とタッチ入力                 |
| **Wireless**        | CC1101 RFトランシーバモジュール (433MHz)           | 無線通信                         |
| **Power Supply**    | 外部3.3V電源                                       | Picoと各モジュールへの安定した電力供給 |

> **[!] 注意**
> Pico本体の3.3V出力ピンは、複数モジュールを同時に駆動するには電流容量が不足する可能性があるため、外部の安定した3.్రాV電源を使用することを強く推奨します。

## 配線 (Wiring)

このプロジェクトでは、ディスプレイとタッチパネルがSPI0バスを、CC1101がSPI1バスをそれぞれ使用します。
配線は以下の表を参考にしてください。

| 機能分類            | Pico Pin | GPIO | 接続先 (モジュール)   | 役割                                         |
| ------------------- | -------- | ---- | ------------------- | -------------------------------------------- |
| **電源**            | **-**    | -    | VCC                 | **外部3.3V電源**に接続                     |
|                     | **GND**  | -    | GND                 | **外部電源のGND**と共通接地にする          |
| **SPI0 (共用)**     | 4        | GP2  | SCK / CLK / T_CLK   | SPIクロック (ディスプレイ/タッチ共用)        |
|                     | 5        | GP3  | SDI / MOSI / T_DIN  | データ線 (Pico -> 機器)                    |
|                     | 6        | GP4  | SDO / MISO / T_DO   | データ線 (機器 -> Pico)                    |
| **ディスプレイ (SPI0)** | 7        | GP5  | CS (Display)        | チップセレクト (ディスプレイ)                |
|                     | 9        | GP6  | DC / RS             | データ/コマンド 切り替え                   |
|                     | 10       | GP7  | RST / RESET         | リセット信号                                 |
|                     | 11       | GP8  | LED / BL            | バックライト制御                             |
| **タッチ (SPI0)**   | 12       | GP9  | T_CS (Touch)        | チップセレクト (タッチ)                      |
|                     | 14       | GP10 | T_IRQ (Touch)       | 割り込み信号                                 |
| **SPI1 (CC1101)**   | 19       | GP14 | SCK                 | SPIクロック (CC1101)                         |
|                     | 20       | GP15 | MOSI                | データ線 (Pico -> CC1101)                    |
|                     | 16       | GP12 | MISO / GDO1         | データ線 (CC1101 -> Pico)                    |
|                     | 17       | GP13 | CSN / CS            | チップセレクト (CC1101)                      |
|                     | 21       | GP16 | GDO0                | 割り込み/データ受信通知                      |

## セットアップと書き込み (Setup & Flash)

### 1. 準備

-   [Go](https://go.dev/doc/install)（1.23以上）と [TinyGo](https://tinygo.org/getting-started/install/) をインストールします。
-   このリポジトリをクローンします。
    ```bash
    git clone https://github.com/kou-tkbys/testing-pico.git
    cd testing-pico
    ```
-   依存関係をダウンロードします。
    ```bash
    go mod tidy
    ```

### 2. ビルドと書き込み

Picoの`BOOTSEL`ボタンを押しながらPCに接続し、以下のコマンドで書き込みます。

-   **Raspberry Pi Pico (RP2040) の場合:**
    ```bash
    tinygo flash -target=pico main.go
    ```

-   **Raspberry Pi Pico 2 (RP2350) の場合:**
    ```bash
    tinygo flash -target=pico2 main.go
    ```

## プログラムの機能詳細 (Functionality)

### タッチパネルのキャリブレーション

タッチの座標がずれている場合、キャリブレーションが必要です。
プログラムはタッチされたRaw座標 (`Raw(X,Y)`) を画面にログ出力します。

1.  ディスプレイの**左上**の角をタッチし、ログに出力される`Raw(X,Y)`の値をメモします。これが `rawMinX`, `rawMinY` になります。
2.  ディスプレイの**右下**の角をタッチし、ログの値をメモします。これが `rawMaxX`, `rawMaxY` になります。
3.  `main.go` 内の以下の定数を、メモした値に書き換えてください。

    ```go
    // main.go L.200あたり
    const (
        rawMinX = 300  // ← ここを手順1のX値に
        rawMaxX = 3800 // ← ここを手順2のX値に
        rawMinY = 200  // ← ここを手順1のY値に
        rawMaxY = 3700 // ← ここを手順2のY値に
    )
    ```

もしタッチした際のX軸やY軸が反転している場合は、`mapRange` 関数の呼び出し部分で出力範囲を逆にすることで修正できます。（例: `0, 240` -> `240, 0`）

## ディレクトリ構成 (Project Structure)

```
testing-pico/
├── cc1101/         # CC1101無線モジュール用ドライバ
├── display/        # ディスプレイ(ILI9341)制御用パッケージ
├── xpt2046/        # タッチパネル(XPT2046)制御用ドライバ
├── main.go         # メインプログラム
├── go.mod          # Goモジュール定義
└── README.md       # このファイル
```

## ライセンス (License)

このプロジェクトは [MIT License](LICENSE) のもとで公開されています。