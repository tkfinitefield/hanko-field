# hanko-field
このレポジトリは、Hanko Field のフロントエンド、バックエンド、管理画面を管理するためのものです。

全体の設計は `doc/design.md` を参照してください。

## フォルダ構成

```
/
├── app/ # Flutterアプリ
├── api/ # Go API
├── admin/ # 管理画面
├── doc/ # ドキュメント
├── web/ # Go Web
```

## 管理画面(admin)
- Go
- htmx
- Tailwind CSS

## バックエンド(api)
- Go
- Cloud Run
- Firestore
- Firebase Auth
- Firebase Storage

## ウェブ(web)
- Go
- htmx
- Tailwind CSS

## タスクについて
- タスクが終了したらチェックを入れること。
