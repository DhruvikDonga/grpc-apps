# GRPC Message Distributed Service

This guide explains how to set up a Go-based service with two APIs:

1. One to fetch all messages across pods.
2. Another to save messages in memory.

We will cover two scenarios:

- How two local Go apps communicate and display all messages over gRPC.
- How to achieve this communication in Kubernetes (K8s).

The service uses one-directional gRPC streaming (response streaming) to stream all messages stored in memory in the Go app.

## 1. Setup gRPC with Protobuf

To define the gRPC service, we use Protocol Buffers (Protobuf). Generate Go code from the `.proto` file using the following command:

```bash
protoc --go_out=. --go-grpc_out=. proto/messages.proto
```

## 2. Local Setup

- **APP 1**: Exposes ports `8081` (HTTP) and `9091` (gRPC).
- **APP 2**: Exposes ports `8082` (HTTP) and `9092` (gRPC).

### 3. Load Image into Kubernetes Cluster

To load the Docker image into the cluster, use the following command:

```bash
kind load docker-image grpc-app:latest --name <cluster-name>
```

## 4. Service Configuration

In Kubernetes, we create a **headless service** to allow direct communication between individual pods. A headless service doesn't allocate a cluster IP for load balancing but allows clients to communicate directly with the pods.

We have two apps: one for HTTP and the other for gRPC. The configuration steps are as follows:

### Key Features of a Headless Service:

- **No ClusterIP**: Setting `clusterIP: None` means no load-balancing IP is assigned, and traffic is routed directly to individual pods.
- **DNS Resolution**: Kubernetes creates DNS entries for each pod in the headless service, enabling direct connections using DNS queries like `pod-name.grpc-service.default.svc.cluster.local`.
- **StatefulSets**: Headless services are often used with StatefulSets where each pod needs to be addressed individually by its DNS name.
- **Pod IPs**: Instead of routing traffic to a single service IP, clients query DNS and get back a list of pod IPs.

### Example Headless Service Configuration:

#### `grpc-service.yaml`

```yaml
apiVersion: v1
kind: Service
metadata:
  name: grpc-service
  labels:
    app: grpc-service
spec:
  clusterIP: None # Headless service (no load balancing, direct pod communication)
  selector:
    app: grpc-service
  ports:
    - name: grpc-port # Name for the gRPC port
      protocol: TCP
      port: 9091 # gRPC port
      targetPort: 9091 # Port on the pod to forward to
```

## 5. Deployment Configuration

Within Kubernetes, services are resolved using the DNS convention:

```
<service-name>.<namespace>.svc.cluster.local
```

This format is standard for referring to services in a Kubernetes cluster.

### Fetching Pod IPs:

To enable gRPC streaming, our function `GetPODIPs()` fetches the IP addresses of all pods associated with the service. These IPs are then used to stream messages.

```go
func GetPODIPs() []string {
    // Logic to fetch pod IPs
}
```

## 6. Example Output in Kubernetes

When fetching the messages across pods, the result would look like this:

```json
[
  {
    "room": "cool_room",
    "client_name": "cool_client_pod_3_k8s",
    "message": "COOOL MESSAGE"
  },
  {
    "room": "cool_room",
    "client_name": "cool_client_pod_2_k8s",
    "message": "COOOL MESSAGE"
  },
  {
    "room": "cool_room",
    "client_name": "cool_client_pod_1_k8s",
    "message": "COOOL MESSAGE"
  }
]
```

This result demonstrates how messages from multiple pods are fetched and displayed over the gRPC stream.

---

By following this setup, we ensure that Go apps can communicate across pods within a Kubernetes environment using gRPC streaming, while also managing the flow of messages directly from each pod in memory.

## Using loadbalancer or ingress

### Ingress

`ingress.yaml` sets up a nginx ingress controller over `grpc-http-service.yaml` to loadbalance in local you can update hosts file and add url grpc-app.lc
Ingress controller:- `kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/cloud/deploy.yaml`

> Note: 
> if you are using headless clusterip instead of grpc-http-service.yaml nginx ingress controller will resolve dns like we are doing in our app main.go for grpc it will do for http 8081 but its http and we don't need to but good for websocket and stateful apps

### Loadbalancer

Use metallb loadbalancer for kind `metallbconfig.yaml` has appropriate commands for it and `loadbalancer.yaml` implements it . Loadbalancer will use the selector and get pods .

# Why not using a database instead of storing in memory .

There are scenarios where in memory becomes a necessity primary one is websockets my side project [simplysocket](https://github.com/DhruvikDonga/simplysocket) which is websocket . A websocket upgrades clients https session to receive and send message its like a bridge which actual sense connects to the app . If the app or a pod is loosed client disconnects from websocket itself .
Consider `client_name` as ws connection it self .

It means 2 things here :-

- Pods can be scaled but clients logic won't be same as normal http request does client will connect to it .
- Pods down scaling is a challenge as down scale will empty all data like in grpc-apps we can see message will be lost if deployment replicas are reduced .

Each pod will have clients in it which are connected . There is a option to use a pub/sub module but why to rely on external service if something can be solved internally if tried 🌟 .
