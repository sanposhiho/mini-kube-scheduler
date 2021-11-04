# score plugin

このブランチでは[filter plugin ブランチ](https://github.com/sanposhiho/mini-kube-scheduler/tree/filter-plugin)の実装に score plugin のサポートを実装します。
また、このブランチでは自身でプラグインを作成する方法も解説します。

Score Plugin は残った候補の Node をスコアリングする拡張点です。

例えば…
・全体の Node のリソースの使用量のバランスがちょうど良くなる Node を優先
・Pod を実行するコンテナイメージをすでに持っている Node を優先
などのスコアリング処理が入るのがこの拡張点になります。

## Score Plugin の実行

実装は[こちら](/minisched/minisched.go#L51-L56)

RunScorePlugins を実行することで Score Plugin を実行しています。

### RunScorePlugins

ということで RunScorePlugins が Score Plugin 実行の本体になっています。

この関数では以下の流れで Score Plugin の実行を行っています。

1. 候補の Node をループでまわす
2. Scheduler に登録された Score Plugin を順にループで回してその Node に対して実行
3. 結果(スコア)を map に格納
4. リストの形にして返却

## NodeNumber Plugin

このブランチでは単純化するために`NodeNumber`というプラグインのみを有効にしています。これは実際の Scheduler の中に実装されている Score Plugin **ではありません**。実装をこのブランチで行っています。

動作は「Node の名前の最後の数字が Pod の名前の最後の数字と一致する Node に高い点数をつける」というものになります。それではこのプラグインを実装することを通して、プラグインの作成の仕方を学んでいきましょう。

### NodeNumber Plugin の実装

基本的に Score Plugin に限らず、Plugin は Scheduling Framework が用意している interface を満たす構造体を作成することで実装したことになります。

```go
// ScorePlugin is an interface that must be implemented by "Score" plugins to rank
// nodes that passed the filtering phase.
type ScorePlugin interface {
	Plugin
	// Score is called on each filtered node. It must return success and an integer
	// indicating the rank of the node. All scoring plugins must return success or
	// the pod will be rejected.
	Score(ctx context.Context, state *CycleState, p *v1.Pod, nodeName string) (int64, *Status)

	// ScoreExtensions returns a ScoreExtensions interface if it implements one, or nil if does not.
	ScoreExtensions() ScoreExtensions
}
```

https://github.com/kubernetes/kubernetes/blob/4c659c5342797c9a1f2859f42b2077859c4ba621/pkg/scheduler/framework/interface.go#L413

ではこちらの interface を満たすように Plugin を作成していきましょう。

実装は[こちら](https://github.com/sanposhiho/mini-kube-scheduler/blob/score-plugin/minisched/plugins/score/nodenumber/nodenumber.go)

`ScoreExtensions`は Normalize Score という別の拡張点で使用する関数なのでここではあまり気にする必要はありません。nil を返しておきましょう。

Score 拡張点のメインロジックは`Score`関数になります。

- Pod の名前の最後の数字を取得 (例：　 pod1 → 1)
- Node の名前の最後の数字を取得 (例：　 node1 → 1)
- 両者が一致していた場合 10 点を返す。一致していなかった場合 0 点を返す。

という実装を行っています。これにより、先程紹介した`NodeNumber`プラグインの仕様を満たしています。

## 動作を試す

このブランチの Scheduler を試す方法は master ブランチの README を参照してください。([ref](https://github.com/sanposhiho/mini-kube-scheduler/blob/master/README.ja.md))

node0~node9 を作成したのち、pod1 と pod3 を作成するシナリオになっています。`NodeNumber`プラグインによって pod1 が node1 へ pod3 が node3 へ bind されるはずですね。
