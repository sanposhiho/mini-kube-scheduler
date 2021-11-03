# initial random scheduler

このブランチではファーストステップとして完全にランダムな Node に Pod を bind する Scheduler を作成します。
このブランチの Scheduler を試す方法は master ブランチの README を参照してください。([ref](https://github.com/sanposhiho/mini-kube-scheduler/blob/master/README.ja.md))

ここで実装しなければいけないフローは以下のようになります。

1. Schedule されていない Pod を見つける
2. その Pod の Schedule を始める
3. 全ての Node を取得する
4. ランダムに Node を一つ選ぶ
5. 選んだ Node に対して Pod を Bind する
6. 1 に戻る

## 1. Schedule されていない Pod を見つける

Scheduler は内部の Queue にスケジュールされていない Pod を貯め、そこから Pod を一つずつスケジュールしていくという仕組みになっています。
そのため、“Schedule されていない Pod を見つけて Queue に貯める仕組み” が必要になります。

### Schedule されていない Pod を貯める仕組みを作る

Queue はとりあえず Slice でシンプルに実装することにします。以下の二つのメソッドを Queue に実装します。

- Queue に Pod を追加する
- Queue から Pod を取り出す

実装は[こちら](/minisched/queue/queue.go#L20-L36)

また、Queue から Pod を取り出すメソッドは「Queue に Pod が一つも存在しない時は Pod が追加されるまで待機する」という実装になっています。

### Schedule されていない Pod を見つける

EventHandler を使用して Schedule されていない Pod を見つけます。 Schedule されていない Pod が見つかった際には前述の Queue に Pod を追加するメソッドを使用して、Pod を Queue に追加します。

実装は[こちら](minisched/eventhandler.go#L16-L32)

### 2. その Pod の Schedule を始める

Scheduler は`scheduleOne`という関数を無限に実行し続けることで Queue から Pod を取り出し、スケジュールするということを繰り返しています。

すなわち、`scheduleOne`が Scheduler のロジックの本体と言えます。

`scheduleOne`ではまず、Queue からスケジュールされていない Pod を 1 つ取り出し Schedule を開始します。ここでは先程の Queue から Pod を取り出すメソッドを使用します。
実装は[こちら](/minisched/minisched.go#L24)

`scheduleOne`は無限に実行されますが、前述のように Queue に Pod が一つも存在しない場合は Queue から Pod を取り出すタイミングで待機が発生します。

### 3. 全ての Node を取得する

実装は[こちら](/minisched/minisched.go#L28)

このスケジューラーでは全ての Node を API から取得しています。
(実際のスケジューラーでは SchedulingCache から(もっと正確にいうと snapshot から)Node を取得しています。)

### 4. ランダムに Node を一つ選ぶ

実装は[こちら](/minisched/minisched.go#L37)

3 で取得した Node からランダムに一つの Node を選択します。

### 5. 選んだ Node に対して Pod を Bind する

実装は[こちら](/minisched/minisched.go#L39)。

ここで Pod を Bind して一つの Pod のスケジュールが終了します。

前述のように`scheduleOne`は無限に実行されているので、5 が終わった時点で次の Pod を Queue から取得し、スケジュールが開始することになります。
