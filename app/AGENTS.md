## Feature-first MVVM directory structure
### `lib/` サブディレクトリ
- `main.dart` : アプリエントリポイント。`ProviderScope` とルート Navigator を初期化。
- `core/` : グローバルな基盤コード。
  - `app/` : アプリ全体設定（AppState 等）。
  - `data/` : DB・API クライアントなどの下位サービス。
  - `localization/` : 多言語テキスト管理。
  - `logging/` : `logger` ライブラリの初期化。
  - `routing/` : `AppRouteConfiguration` や `AppTab` などの Navigator 2.0 ルーティング定義。
  - `services/`/`utils/` : レイヤ共通ユーティリティ。
- `shared/` : 各機能から再利用する UI コンポーネントや共通コントローラ。
  - `widgets/`, `controllers/`, `notification/`, `security/`, `utils/` に機能別分類。
- `features/` : ドメイン機能ごとのモジュール。Feature + MVVM で `application/`（状態・ViewModel）, `data/`（リポジトリ・モデル）, `domain/`（エンティティ）, `presentation/`（UI）を分割。

## アーキテクチャ
- Feature + MVVM パターンを採用する。
- Navigation
  - 画面遷移には Navigator 2.0 を用いる。
  - できる限り Navigator.push や pushNamed は使わず、 Navigator 2.0 の仕組みを用いる。
- Riverpod 3.0
  - 自動コード生成は用いない。
  - Provider, FutureProvider や StreamProvider は使用しない。代わりに Notifier, AsyncNotifier を使用する。
  - ProviderScope は ルートに一つ配置する。ルート以外の箇所に ProviderScope を配置してはならない。
- `print` は使わずに `logger` ライブラリを使用する。
- できる限りコメントを日本語でつけること
- 時刻を取得する際は clock ライブラリを使用して、テストを容易にする。

## Riverpod 3.0
Use Riverpod 3.0 for business logic.

### Providers
Providers come 6 variants but we only use Notifier and AsyncNotifier:

| | Synchronous | Future | Stream |
| --- | --- | --- | --- |
| Unmodifiable | Provider (DO NOT USE) | FutureProvider (DO NOT USE) | StreamProvider (DO NOT USE) |
| Modifiable | NotifierProvider | AsyncNotifierProvider | StreamNotifierProvider |

Only use Notifier and AsyncNotifier. Do not use Provider, FutureProvider, and StreamProvider.

### Ref.mounted
You can use ref.mounted to check if a provider is still mounted after an async operation:

```dart
class TodoList extends Notifier<List<Todo>> {
  @override
  List<Todo> build() => [];

  Future<void> addTodo(String title) async {
    // Post the new todo to the server
    final newTodo = await api.addTodo(title);
    // Check if the provider is still mounted
    // after the async operation
    if (!ref.mounted) return;

    // If it is, update the state
    state = [...state, newTodo];
  }
}
```

### autoDispose and family
```dart
final provider = NotifierProvider.autoDispose<MyNotifier, int>(
  MyNotifier.new,
);

class MyNotifier extends Notifier<int> {
  @override
  int build() => 0;
}
```

```dart
final provider = NotifierProvider.family<CounterNotifier, int, Argument>(
  CounterNotifier.new,
);

class CounterNotifier extends Notifier<int> {
  CounterNotifier(this.arg);
  final Argument arg;
  @override
  int build() => 0;
}
```

```dart
final provider = AsyncNotifierProvider.autoDispose<MyNotifier, int>(
  MyNotifier.new,
);

class MyNotifier extends AsyncNotifier<int> {
  @override
  Future<int> build() async => 0;
}
```

```dart
final provider = AsyncNotifierProvider.family<CounterNotifier, int, Argument>(
  CounterNotifier.new,
);

class CounterNotifier extends AsyncNotifier<int> {
  CounterNotifier(this.arg);
  final Argument arg;
  @override
  Future<int> build() async => 0;
}
```

Instead of Notifier+FamilyNotifier+AutoDisposeNotifier+AutoDisposeFamilyNotifier, we always use the Notifier class.
Instead of AsyncNotifier+AsyncFamilyNotifier+AsyncAutoDisposeNotifier+AsyncAutoDisposeFamilyNotifier, we always use the AsyncNotifier class.

### Mutations
A new feature called "mutations" is introduced in Riverpod 3.0.
This feature solves two problems:

It empowers the UI to react to "side-effects" (such as form submissions, button clicks, etc), to enable it to show loading/success/error messages. Think "Show a toast when a form is submitted successfully".
It solves an issue where onPressed callbacks combined with Ref.read and Automatic disposal could cause providers to be disposed while a side-effect is still in progress.
The TL;DR is, a new Mutation object is added. It is declared as a top-level final variable, like providers:

```dart
final addTodoMutation = Mutation<void>();

After that, your UI can use ref.listen/ref.watch to listen to the state of mutations:

class AddTodoButton extends ConsumerWidget {
  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Listen to the status of the "addTodo" side-effect
    final addTodo = ref.watch(addTodoMutation);

    return switch (addTodo) {
      // No side-effect is in progress
      // Let's show a submit button
      MutationIdle() => ElevatedButton(
        // Trigger the side-effect on click
        onPressed: () {
          // TODO see explanation after the code snippet
        },
        child: const Text('Submit'),
      ),
      // The side-effect is in progress. We show a spinner
      MutationPending() => const CircularProgressIndicator(),
      // The side-effect failed. We show a retry button
      MutationError() => ElevatedButton(
        onPressed: () {
          // TODO see explanation after the code snippet
        },
        child: const Text('Retry'),
      ),
      // The side-effect was successful. We show a success message
      MutationSuccess() => const Text('Todo added!'),
    };
  }
}
```

Last but not least, inside our onPressed callback, we can trigger our side-effect as followed:

```dart
onPressed: () {
  addTodoMutation.run(ref, (tsx) async {
    // This is where we run our side-effect.
    // Here, we typically obtain a Notifier and call a method on it.
    await tsx.get(todoListProvider.notifier).addTodo('New Todo');
  });
}
```

note
Note how we called tsx.get here instead of Ref.read.
This is a feature unique to mutations. That tsx.get obtains the state of a provider, but keep it alive until the mutation is completed.

## test
### test double naming convention
mock: 振る舞いを検証するためのテストダブル。呼び出し回数や引数など「どう使われたか」をアサートしたいときに使います。
stub: 戻り値を差し替えるだけのシンプルなテストダブル。呼び出された事実は気にせず、「こう返ってくる」と決め打ちしたい場合に使います。
fake: 実装を簡略化した軽量版。メモリ内の擬似リポジトリなど、小規模でも本物に近い振る舞いを提供したいときに使います。

### test directory structure
### `test/` サブディレクトリ
- `core/`, `shared/`, `features/` など `lib/` と同じ構成で配置。ユースケース・ViewModel・サービスの検証を目的としたテストコードを格納。

### test double directory structure
- `test/doubles/mocks/` : mock objects
- `test/doubles/stubs/` : stub objects
- `test/doubles/fakes/` : fake objects

### clock, fake_async
テストでは、時間の経過を制御するために clock と fake_async を使用します。
