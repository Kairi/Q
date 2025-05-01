# q: ChatGPT CLI インターフェース

`q` は OpenAI の ChatGPT API を活用したコマンドライン対話ツールです。会話履歴をローカルに保存し、継続的な対話が可能です。

## 動作要件
- Go 1.24 以上（ソースからビルドする場合）
- 環境変数 `OPENAI_API_KEY` に OpenAI API キーを設定
- インターネット接続

## インストール

### ソースからビルド
```bash
git clone https://github.com/Kairi/q.git
cd q
go build -o q main.go
```

または、`go install` でインストール:
```bash
go install github.com/Kairi/q@latest
```
（インストール先: `$GOPATH/bin/q`）

## 使い方

```bash
q [--model MODEL] [--system SYSTEM_PROMPT] [--thread THREAD_ID]
```

- `--model`：使用するモデル（デフォルト: `o4-mini`）。例: `gpt-4`, `gpt-3.5-turbo`
- `--system`：システムプロンプト（スレッド未作成時のみ適用）
- `--thread`：会話スレッド ID（デフォルト: `default`）。同一 ID を指定すると既存履歴をロードして続きの対話が可能

### 環境変数
- `OPENAI_API_KEY`：API キーを設定。必須。

```bash
export OPENAI_API_KEY=sk-...
```

### 対話例

```bash
$ q --model gpt-4 --system "あなたは親切なアシスタントです。" --thread mythread
ChatGPT CLI interactive chat (gpt-4) [thread: mythread]
System: あなたは親切なアシスタントです。

🤖 ChatGPT: はい、どういったご用件でしょうか？

Type your message and press Enter. Type 'exit' or Ctrl+D to quit.
You: こんにちは！
ChatGPT is thinking...
🤖 ChatGPT: こんにちは！今日はどんなお手伝いが必要ですか？

You: exit
Exiting.
```

## 会話履歴の保存場所
会話履歴は JSON 形式で以下に保存されます:

- Linux/macOS: `~/.config/chat-cli/threads/<THREAD_ID>.json`
- Windows: `%APPDATA%\\chat-cli\\threads\\<THREAD_ID>.json`

## 注意事項
- 既存スレッドをロードすると `--system` プロンプトは無視され、警告が表示されます。
- セッション中に異常終了した場合、`.tmp` ファイルが残る可能性があります。
- 配布バイナリ `q` は `.gitignore` に含まれるため、通常はリポジトリにコミットされません。

## ライセンス
ライセンス情報を追記してください（必要に応じて）。