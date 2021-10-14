# mini-kube-scheduler: Scheduler for learning Kubernetes Scheduler

**このスケジューラーは本番環境での使用を想定して作成されたものではありません**

Hello world.

This is mini-kube-scheduler -- the scheduler for learning Kubernetes Scheduler.

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
