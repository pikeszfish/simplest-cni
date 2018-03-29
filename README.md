# Simplest CNI (最简单的 CNI)
As I was inspired by the article [也许您的Kubernetes集群并不需要SDN](https://jishu.io/kubernetes/your-kubernetes-cluster-may-not-need-sdn/), I found k8s CNI can just use the pod CIDR auto-allocated by the `kube-controller-manager` for each node with `host-local` from [containernetworking/plugins](https://github.com/containernetworking/plugins) rather than other IPAM plugins which use distributed stores(i.e etcd) like in Calico or flannel.

## Feature
- [x] CNI 0.3.0
- [x] auto rotate config as ConfigMap changed

## Installation
```bash
kubectl apply -f https://raw.githubusercontent.com/pikeszfish/simplest-cni/master/k8s/simplest-cni.yml
```

## Build 
```bash
docker build -t pikeszfish/simplest-cni:v0.1.0 .
```

## Reference
- [也许您的Kubernetes集群并不需要SDN](https://jishu.io/kubernetes/your-kubernetes-cluster-may-not-need-sdn/)
- [kubeup/hostroutes](https://github.com/kubeup/hostroutes)
- [install-cni.sh from projectcalico/cni-plugin](https://github.com/projectcalico/cni-plugin)

## License
simplest-cni is available under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
