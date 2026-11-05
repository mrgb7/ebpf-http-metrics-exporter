CLUSTER := ebpf-demo
# Always talk to the k3d cluster, never whatever context happens to be active.
KUBECTL := kubectl --kubeconfig $$(k3d kubeconfig write $(CLUSTER))

.PHONY: up cluster build deploy metrics ui down

## up: create the cluster, build images and deploy everything
up: cluster build deploy

# One node on purpose: k3d nodes are containers sharing the same kernel,
# so a second agent would count the same traffic twice.
cluster:
	k3d cluster create $(CLUSTER) --agents 0 --wait

build:
	docker build --load -t demo-app:local app
	docker build --load -t http-agent:local agent
	docker save demo-app:local http-agent:local -o /tmp/ebpf-demo-images.tar
	k3d image import -c $(CLUSTER) /tmp/ebpf-demo-images.tar

deploy:
	$(KUBECTL) apply -f deploy/
	$(KUBECTL) -n default rollout status deploy/orders deploy/shop
	$(KUBECTL) -n kube-system rollout status ds/http-agent

## metrics: print the agent's HTTP response counters
metrics:
	@pod=$$($(KUBECTL) -n kube-system get pods -l app=http-agent -o jsonpath='{.items[0].metadata.name}'); \
	$(KUBECTL) get --raw "/api/v1/namespaces/kube-system/pods/$$pod:9100/proxy/metrics" | grep '^http_responses_total'

## ui: open Prometheus on http://localhost:9090
ui:
	$(KUBECTL) -n default port-forward deploy/prometheus 9090:9090

down:
	k3d cluster delete $(CLUSTER)
