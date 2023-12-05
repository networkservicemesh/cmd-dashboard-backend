# Network Service Mesh Dashboard Backend

NSM dashboard backend part provides graphical model data for [the UI part](https://github.com/networkservicemesh/cmd-dashboard-ui) through the REST API

Written in [Go](https://go.dev/)

The entire NSM dashboard deployment info see [here](https://github.com/networkservicemesh/deployments-k8s/tree/main/examples/observability/dashboard)

## Dev/debug

### To run dashboard backend in the cluster:

1. `git clone git@github.com:networkservicemesh/deployments-k8s.git`
2. `cd deployments-k8s/examples/observability/dashboard`
3. Edit `dashboard-pod.yaml` and remove the `dashboard-ui` container
4. `kubectl apply -f dashboard-pod.yaml`
5. `kubectl apply -f dashboard-backend-service.yaml`
6. `kubectl port-forward -n nsm-system service/dashboard-backend 3001:3001`
7. Check `http://localhost:3001/nodes` in the browser

### To run dashboard backend with a custom container ([Docker](https://docs.docker.com/engine/install/) have to be installed) in the cluster:

1. `git clone git@github.com:networkservicemesh/cmd-dashboard-backend.git`
2. `cd cmd-dashboard-backend`
3. Make the necessary code changes
4. Create a [Dockerhub](https://hub.docker.com/) repository
5. `docker build -t your-dh-namespace/dh-repo-name .`
6. `docker push your-dh-namespace/dh-repo-name`
7. `cd deployments-k8s/examples/observability/dashboard`
8. Edit `dashboard-pod.yaml` and set your Dockerhub image address for the `dashboard-backend` container
9. Execute the steps 3-7 from the previous section
