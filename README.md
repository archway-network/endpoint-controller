# Kubernetes Endpoint Controller

Kubernetes Endpoint Controller is a program written in Golang that watches Kubernetes service annotations and creates Kubernetes endpoints based on those annotations.
This program is meant to watch Cosmos blockchain nodes and determine their health and dynamically remove/add endpoint targets.

Controller will generate Endpoint with correct targets and port assignments based on annotations and service resource.

## Features
- HTTP health check on endpoints
- GRPC health check on endpoints
- Blockchain node falling behind

## Installation

### Prerequisites

- Kubernetes cluster version 1.16 or later
- `kubectl` version 1.16 or later
- Golang version 1.20 or later
- Docker version 20.10 or later

### Deployment

1. Clone the repository:
```bash
git clone https://github.com/example/endpoint-controller.git
```
2. Deploy the example
```
kubectl apply -f k8s
```

#### Configuration
Controller uses environment variables to configure how it behaves
| Variable  | Description | Default 
---         | ---         | --- 
SYNC_PERIOD | Reconcile period in seconds| 30
BLOCK_MISS  | Allowed missed blocks amount | 6

## Usage
1. Annotate a service with the required endpoints information\
`endpoint-controller-enable` is set to `true` \
`endpoint-controller-addresses` is a list of Endpoint target IPs
```
  annotations:
    endpoint-controller-enable: "true"
    endpoint-controller-addresses: "1.1.1.1,2.2.2.2,3.3.3.3"
```
2. Check that the controller has created the endpoint
```
kubectl get endpoints my-service
```

## Development
### Build
```
go build -o bin/endpoint-controller .
```
or

```
docker build -t endpoint-controller:latest .
```
```
