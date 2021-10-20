# mini-kube-scheduler: Scheduler for learning Kubernetes Scheduler

**このスケジューラーは本番環境での使用を想定して作成されたものではありません**

Hello world.

このスケジューラーは学習向けに開発されたもので、コマンドライン上から動作を確認しつつ、スケジューラーの機能の実装の様子を見て取ることができます。

また、client-goを用いてシナリオを記述し、スケジューラーの動作を試すことができます。

```go
func scenario(client clientset.Interface) error {
	ctx := context.Background()

	// create node0 ~ node9, but all nodes are unschedulable
	for i := 0; i < 9; i++ {
		suffix := strconv.Itoa(i)
		_, err := client.CoreV1().Nodes().Create(ctx, &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node" + suffix,
			},
			Spec: v1.NodeSpec{
				Unschedulable: true,
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create node: %w", err)
		}
	}

	klog.Info("scenario: all nodes created")

	_, err := client.CoreV1().Pods("default").Create(ctx, &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod1"},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "container1",
					Image: "k8s.gcr.io/pause:3.5",
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("create pod: %w", err)
	}

	klog.Info("scenario: pod1 created")

	// wait to schedule
	time.Sleep(3 * time.Second)

	pod1, err := client.CoreV1().Pods("default").Get(ctx, "pod1", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get pod: %w", err)
	}

    klog.Info("scenario: pod1 is bound to " + pod1.Spec.NodeName)
	
	return nil
}
```

## スケジューラーの進化を追う

このスケジューラーはブランチごとに段階を踏んで進化していく様子を追うことができるように開発されています。
それぞれのブランチでスケジューラーの動作を確認し、コードを変更するなどして理解を深めることができます。

TODO: 各ブランチにそのブランチで追加された機能の説明を加える。

1. [initial scheduler](https://github.com/sanposhiho/mini-kube-scheduler/tree/initial-random-scheduler)

最も初期の段階のスケジューラーであり、全てのNodeから完全にランダムにPodを配置します。

2. [filter plugins](https://github.com/sanposhiho/mini-kube-scheduler/tree/filter-plugin)

このブランチのスケジューラーはfilter pluginの実行に対応しています。unschedulable nodeプラグインのみが有効になっています。

3. [score plugins](https://github.com/sanposhiho/mini-kube-scheduler/tree/score-plugin)

このブランチのスケジューラーはさらにscore pluginの実行に対応しています。custom pluginであるnodenumberプラグインのみがscore pluginとして有効になっています。nodenumber プラグインの実装もブランチ内に存在しています。

4. [prescore plugins](https://github.com/sanposhiho/mini-kube-scheduler/tree/prescore-plugin)

このブランチのスケジューラーはさらにpre-score pluginの実行に対応しています。custom pluginであるnodenumberプラグインにpre-score pluginの機能を実装し、有効にしています。

5. [permit plugins](https://github.com/sanposhiho/mini-kube-scheduler/tree/permit-plugin)

このブランチのスケジューラーはさらにpremit pluginの実行に対応しています。custom pluginであるnodenumberプラグインにpremit pluginの機能を実装し、有効にしています。
また、このブランチでは、Binding Cycleが並行にgoroutineで実行されるようになっています。

6. [scheduling queue](https://github.com/sanposhiho/mini-kube-scheduler/tree/scheduling-queue)

このブランチにはScheduling Queueが実装されています。また、スケジュールの失敗時に、PodをunschedulableとしてQueueに戻すことにも対応しています。

7. [eventhandler](https://github.com/sanposhiho/mini-kube-scheduler/tree/event-handler)

このブランチにはEventHandlerを用いたQueueの更新に対応しています。これにて、スケジュールに失敗したPodの再スケジュールに対応しています。


# スケジューラーのメインロジックに関して

スケジューラーのメインロジックは、[/minisched](./minisched)に存在しています。
このディレクトリ以下に変更を加えることで、起動するスケジューラーのロジックを変更することができます

プログラムの実行時、このスケジューラーは[ここ](/scheduler/scheduler.go#L50-L80)で起動されます。
そのため、起動時の振る舞いを変更したい場合や、何か変数を渡したい場合などはこちらから変更してください。

# シナリオ

[ここ](/sched.go#L70) でclient-goを用いてAPIを叩き、スケジューラーの動作を試すシナリオが記述されています。
自由に変更し、スケジューラーの動作確認に使用してください。

# シナリオ/スケジューラーの実行に関して

スケジューラーを起動して、シナリオを実行するにはGoとetcdが必要になります
etcdに関してはこちらのKubernetes本体に用意されているインストール方法を使用できます。

[kubernetes/kubernetes/hack/install-etcd.sh](https://github.com/kubernetes/kubernetes/blob/master/hack/install-etcd.sh).

もちろん別の方法でinstallしていただいても構いません。

そして `make start` でスケジューラーとスケジューラーに必要なコンポーネントが立ち上がり、シナリオが実行されます。

# 参考

このmini-kube-schedulerのスケジューラーの起動やシナリオの実行などの全体的な仕組みに関しては[kubernetes-sigs/kube-scheduler-simulator](https://github.com/kubernetes-sigs/kube-scheduler-simulator)を参考にしております。
