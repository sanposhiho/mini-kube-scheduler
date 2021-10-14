# mini-kube-scheduler: Scheduler for learning Kubernetes Scheduler

**This scheduler is not for production**

[日本語版はこちら(Japanese ver)](/README.ja.md)

Hello world. 

This is mini-kube-scheduler -- the scheduler for learning Kubernetes Scheduler.

# main logic of this scheduler

Most of the codes for this scheduler is placed under [/minisched](./minisched). 
You can change this scheduler to what you want.

And this scheduler is started on [here](/scheduler/scheduler.go#L50-L80)

If you want to change how to start the scheduler, you can change here.

# scenario

You can write scenario [here](/sched.go#L70) and check this scheduler's behaviour.

# How to start this scheduler and scenario

To run this scheduler and start scenario, you have to install Go and etcd.
You can install etcd with [kubernetes/kubernetes/hack/install-etcd.sh](https://github.com/kubernetes/kubernetes/blob/master/hack/install-etcd.sh).

And, `make start` starts your scenario.

# Note

This mini-kube-scheduler starts scheduler, etcd, api-server and pv-controller.

The whole mechanism is based on [kubernetes-sigs/kube-scheduler-simulator](https://github.com/kubernetes-sigs/kube-scheduler-simulator)

