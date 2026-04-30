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
}
