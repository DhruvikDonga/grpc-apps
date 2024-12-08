
# GRPC Message Distributed Service

This guide explains how to set up a Go-based service with two APIs:

1. One to fetch all messages across pods.
2. Another to save messages in memory.

We cover two scenarios:
- How two local Go apps communicate and display all messages over gRPC.
- How to achieve this communication in Kubernetes (K8s).

We use one-directional gRPC streaming (response streaming) to stream all messages stored in memory in the Go app.

## 1. Setup gRPC with Protobuf

To define the gRPC service, we use Protocol Buffers (Protobuf). The following command generates Go code from the `.proto` file:

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

In Kubernetes, we need to create a **headless service** to allow direct communication between individual pods. A headless service does not allocate a cluster IP for load balancing but instead allows clients to communicate directly with the pods.

### Key Features of a Headless Service:
- **No ClusterIP**: Setting `clusterIP: None` means no load-balancing IP is assigned, and traffic is routed directly to the individual pods.
- **DNS Resolution**: Kubernetes automatically creates DNS entries for each pod in the headless service, allowing clients to connect directly using DNS queries like `pod-name.grpc-service.default.svc.cluster.local`.
- **StatefulSets**: Headless services are commonly used with StatefulSets, where each pod needs to be addressed individually by its DNS name.
- **Pod IPs**: Instead of routing traffic to a single service IP, clients can query DNS and get back a list of pod IPs.

### Headless Service Example:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: grpc-service
spec:
  clusterIP: None  # Headless service
  selector:
    app: grpc-service
  ports:
    - name: grpc-port
      port: 9091
      targetPort: 9091
    - name: http-port
      port: 8081
      targetPort: 8081
```

## 5. Deployment Configuration

Within Kubernetes, services are resolved using a DNS convention like:

```
<service-name>.<namespace>.svc.cluster.local
```

This is the standard format to refer to services in a Kubernetes cluster.

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