# q: AI チャット CLI インターフェース

`q` は OpenAI ChatGPT と Google Gemini の両方の API を活用したシンプルなコマンドライン対話ツールです。会話履歴をローカルに保存し、継続的な対話を楽しむことができます。

## 動作要件
- Go 1.23 以上（ソースからビルドする場合）
- API キーの環境変数設定:
  - OpenAI モデル使用時: `OPENAI_API_KEY`
  - Google Gemini モデル使用時: `GEMINI_API_KEY`
- インターネット接続

## インストール

### ソースからビルド
```bash
git clone https://github.com/Kairi/Q.git
cd Q
go build -o q main.go
```

または、`go install` でインストール:
```bash
go install github.com/Kairi/Q@latest
```
（実行ファイルは `$GOPATH/bin/q` にインストールされます）

## 使い方

```bash
q [--model MODEL] [--system SYSTEM_PROMPT]
```

- `--model`：使用するモデル（デフォルト: `gemini-2.5-flash-lite-preview-06-17`）
  - OpenAI モデル: `gpt-5`, `gpt-4o-mini`, `gpt-4`, `gpt-3.5-turbo` など
  - Google Gemini モデル: `gemini-2.5-flash-lite-preview-06-17`, `gemini-pro-1.0` など
- `--system`：システムプロンプト（新しい会話開始時のみ適用）

### 環境変数
使用するモデルに応じて適切な API キーを設定してください：

```bash
# OpenAI モデル使用時
export OPENAI_API_KEY=sk-...

# Google Gemini モデル使用時  
export GEMINI_API_KEY=your-gemini-api-key
```

### 対話例

```console
$ q --model gemini-2.5-flash-lite-preview-06-17 --system "あなたは親切なアシスタントです。"
ChatGPT CLI interactive chat (gemini-2.5-flash-lite-preview-06-17) [thread: default]
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

- Linux/macOS: `~/.config/q/threads/<THREAD_ID>.json`
- Windows: `%APPDATA%\\q\\threads\\<THREAD_ID>.json`

## 注意事項
- 既存の会話履歴がある場合、`--system` プロンプトは無視されます。
- モデル名に `gemini` が含まれている場合は Google Gemini API が使用され、それ以外は OpenAI API が使用されます。
- セッション中に異常終了した場合、`.tmp` ファイルが残る可能性があります。
- 配布バイナリ `q` は `.gitignore` に含まれるため、通常はリポジトリにコミットされません。

## ライセンス
このプロジェクトは MIT ライセンスの下で公開されています。詳細は [LICENSE](LICENSE) ファイルをご覧ください。