# filter plugin

このブランチでは[initial random scheduler ブランチ](https://github.com/sanposhiho/mini-kube-scheduler/tree/initial-random-scheduler)(ランダムにスケジュールを行うスケジューラー)の実装に filter plugin のサポートを実装します。

Scheduler は Scheduling Framework という仕組みに沿って内部的に動作しています。
![Scheduling Framework](https://d33wubrfki0l68.cloudfront.net/4e9fa4651df31b7810c851b142c793776509e046/61a36/images/docs/scheduling-framework-extensions.png)
https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/

Filter phase は Pod を実行できない(したくない)Node を候補から除外する拡張点です。

例えば…
・リソース不足で Pod が実行できない Node
・nodeSelector の条件に一致しない Node
を除外するのがこのポイントになります

## Filter Plugin の実行

実装は[こちら](/minisched/minisched.go#L43-L47)

RunFilterPlugins を実行し、エラーが返ってくるとその Pod のスケジュールを諦めるという実装になっています。

### RunFilterPlugins

ということで RunFilterPlugins が Filter Plugin 実行の本体になっていますね。

この関数では以下の流れで Filter Plugin の実行を行っています。

1. 候補の Node をループでまわす
2. Scheduler に登録された Filter Plugin を順にループで回してその Node に対して実行
3. 2 のループで全ての Filter Plugin から承認された Node を候補の Node リストに入れる
4. 一つも候補が残らなかった際はエラーを返す。一つでも候補が残った際は候補を返す

## NodeUnschedulable Plugin

このブランチでは単純化するために`NodeUnschedulable`というプラグインを有効にしています。これは実際の Scheduler の中に実装されている Filter Plugin の一つです。

動作としては、「Node の.Spec.Unschedulable を見て、true の Node を候補から外す」というものです。

## 動作を試す

このブランチの Scheduler を試す方法は master ブランチの README を参照してください。([ref](https://github.com/sanposhiho/mini-kube-scheduler/blob/master/README.ja.md))

このブランチでは Node0~Node9 を`.Spec.Unschedulable: true`にして作成し、その後 Node10 を`.Spec.Unschedulable`の指定無し(=すなわち Unschedulable ではない)で作成しています。
その後、Pod を作成しており、どの Node に Bind されるかをログに出力します。Node0~Node9 は全て`NodeUnschedulable`プラグインに Filtering され、Node10 に必ず Bind されるはずです。
