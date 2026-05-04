# kakeibo-cli

シンプルな家計簿CLIツール。収支の記録・表示・保存ができます。

## インストール

```sh
git clone https://github.com/yourname/kakeibo-cli.git
cd kakeibo-cli
go build -o kakeibo-cli
mv kakeibo-cli ~/.local/bin/
```

## 使い方

```
kakeibo-cli <コマンド> [引数]
```

| コマンド | 説明 |
|---------|--------------------------|
| `help` | ヘルプを表示 |
| `expense` | 支出を記録 |
| `income` | 収入を記録 |
| `delete` | 記録を削除 |
| `show` | 記録を一覧表示する |
| `stats` | 統計・グラフを表示する |
| `export` | JSONにエクスポート |
| `import` | JSONからインポート |
| `repl` | 対話モードで起動 |

## コマンド例

```sh
# 支出を記録（金額 日付 カテゴリ メモ）
# Food.Lunchのようにドット区切りでサブカテゴリを指定できます。
kakeibo-cli expense 1850 t Food.Lunch ランチ

# 収入を記録
kakeibo-cli income 210000 0425 Salary

# 今年の月ごとの統計を表示
kakeibo-cli stats y

# 今月の日ごとの統計を表示
kakeibo-cli stats m

# 今年のカテゴリ別の統計を表示
kakeibo-cli stats y category

# 最近追加した5件を表示
kakeibo-cli show latest

# バックアップ
kakeibo-cli export
```
