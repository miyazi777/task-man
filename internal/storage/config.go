// Package storage は tasks.yaml と添付ファイルの永続化を担う。Repository
// インターフェースとその唯一の実装 YAMLRepository を提供する。
package storage

// AppConfig は tasks.yaml に保存されるアプリケーション設定をまとめたもの。
// 戻り値の数を抑えつつ、設定追加に対する破壊的変更を局所化する目的で導入。
type AppConfig struct {
	// DataBaseDirectory はタスクの情報格納先ディレクトリ (yamlDir からの相対パス)。
	// 空文字なら yamlDir 直下を使う。
	DataBaseDirectory string

	// Layout はタスクリスト画面のペイン比率設定。
	// 4 値とも nil なら未設定 (=従来計算でレンダリング)。
	Layout LayoutConfig

	// Applications はファイルを開く際に使用できる外部アプリケーションの一覧。
	// 空ならファイルオープナー機能は $EDITOR にフォールバックする。
	Applications []Application

	// FileOpeners は拡張子ごとに使用するアプリケーション (ID 配列) の対応表。
	// 該当拡張子が無ければ $EDITOR にフォールバック。
	FileOpeners []FileOpener
}

// Application は外部起動可能なアプリケーション 1 件分の情報。
//   - ID: yaml 内で一意。FileOpener.ApplicationIDs から参照される。
//   - Name: モーダル表示用ラベル。
//   - Run: 環境変数表記 ("$EDITOR" 等) または PATH 上のコマンド名 ("md-viewer") 等を許容する。
//     起動時に os.ExpandEnv で展開し、空白区切りで引数を分けるので "nvim --noplugin" のような表記も使える。
type Application struct {
	ID   int
	Name string
	Run  string
}

// FileOpener は単一拡張子に対するアプリケーション候補の対応。
//   - Extension: ドット無しの拡張子 (例: "md", "txt")。大文字小文字は無視 (比較時は小文字化)。
//   - ApplicationIDs: 候補となる Application.ID の配列。順序がモーダル表示順 (`o` キー押下時)。
//   - DefaultApp: file list で `enter` 押下時に起動するアプリの Application.ID。
//     0 (= 未指定) なら $EDITOR にフォールバック。
type FileOpener struct {
	Extension      string
	ApplicationIDs []int
	DefaultApp     int
}

// LayoutConfig はタスクリスト画面のペイン比率を保存する設定。
// 各値は 0.0〜1.0 の比率で、nil = 未設定を表す。
//   - TaskListWidth: 画面幅に対するタスクリストの占有比率
//   - TaskDetailHeight / FileListHeight / FilePreviewHeight: 右ペイン縦領域に対する占有比率
//     (3 値の合計が 1.0 になるよう確定保存時に正規化される)
//
// 4 値が揃わないかぎり View 側はフォールバック (従来計算) を使う。
type LayoutConfig struct {
	TaskListWidth     *float64
	TaskDetailHeight  *float64
	FileListHeight    *float64
	FilePreviewHeight *float64
}
