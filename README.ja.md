# prescore plugin

このブランチでは[score plugin ブランチ](https://github.com/sanposhiho/mini-kube-scheduler/tree/score-plugin)の実装に prescore plugin のサポートを実装します。

PreScore 拡張点は Score Plugin の実行のための準備を行う拡張点で、Score Plugin 実行のための事前計算などを行います。

実際に score plugin ブランチで作成した`NodeNumber`プラグインを PreScore 拡張点のサポートを追加してみましょう。(一つのプラグインはこのように複数の拡張点で動作することができます。)

## Cycle state について

一つの Pod の スケジュール でのデータを保存しているものです。
Pod のスケジュール開始ごとに毎回新しく作成されます。

Plugin(等)は CycleState にデータを読み書きすることができます。
使用例として、PreScore や PreFilter といった拡張点では Score や Filter などのプラグインの実行のための事前計算を行い、その結果を Cycle state に保存します。すると、Score/Filter プラグインは実行時に CycleState からそのデータを取り出して利用することができます。

## `NodeNumber`プラグインの PreScore 拡張点のサポート

PreScore Plugin の実装のためには以下の interface を満たす必要があります。

```go
// PreScorePlugin is an interface for "PreScore" plugin. PreScore is an
// informational extension point. Plugins will be called with a list of nodes
// that passed the filtering phase. A plugin may use this data to update internal
// state or to generate logs/metrics.
type PreScorePlugin interface {
	Plugin
	// PreScore is called by the scheduling framework after a list of nodes
	// passed the filtering phase. All prescore plugins must return success or
	// the pod will be rejected
	PreScore(ctx context.Context, state *CycleState, pod *v1.Pod, nodes []*v1.Node) *Status
}
```

https://github.com/kubernetes/kubernetes/blob/4c659c5342797c9a1f2859f42b2077859c4ba621/pkg/scheduler/framework/interface.go#L391

実装は[こちら](/minisched/plugins/score/nodenumber/nodenumber.go#L41)

Pod の末尾の数字を取り出し、CycleState に保存しています。
これは一つの Pod のスケジュール中には Pod の名前の末尾の数字はもちろん変化しないため、毎回`Score`がよばれるたびに Pod の名前から末尾の数字の取り出しを行う処理をすることはほんの少し無駄だからです。

(※ 例としてこの処理を PreScore で行っているだけであり、実際にはそこまでパフォーマンスの向上があるかはわかりません。むしろ、CycleState の処理の方が Pod の末尾の数字を取り出す処理より重いと思いますが、例として何かしら処理を PreScore に移したかったなのでご愛嬌ということで…)

## PreScore の実行のサポート

実装は[こちら](/minisched/minisched.go#L53-L59)

Scheduler に登録されている PreScore Plugin を全て順に実行しているだけです。
