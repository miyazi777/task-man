package storage

// AppConfig は tasks.yaml に保存されるアプリケーション設定をまとめたもの。
// 戻り値の数を抑えつつ、設定追加に対する破壊的変更を局所化する目的で導入。
type AppConfig struct {
	// DataBaseDirectory はタスクの情報格納先ディレクトリ (yamlDir からの相対パス)。
	// 空文字なら yamlDir 直下を使う。
	DataBaseDirectory string

	// Editor はタスクファイルを開く外部エディタのコマンド (yaml の applications.editor)。
	// "$EDITOR" のような環境変数表記を許容し、起動時に os.ExpandEnv で展開する。
	Editor string

	// Layout はタスクリスト画面のペイン比率設定。
	// 4 値とも nil なら未設定 (=従来計算でレンダリング)。
	Layout LayoutConfig
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
