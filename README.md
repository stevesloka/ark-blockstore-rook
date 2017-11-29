# ark-plugin-rook

Ark plugin for Heptio Ark

## Setup

This sample uses Rook for the Block Storage inside Kubernetes. Ark will use Minio for its object storage backend. Additionally, the sample nginx-app is modified to utilize rook storage classes so persistent volumes are created inside Rook. 

1. Deploy Rook (https://rook.io/docs/rook/master/kubernetes.html#quickstart)
2. Deploy Rook Api (`kubectl apply -f https://raw.githubusercontent.com/stevesloka/ark-blockstore-rook/master/rest-api/deployment.yaml`)
3. Deploy Ark Prereqs: (`kubectl apply -f https://raw.githubusercontent.com/heptio/ark/master/examples/common/00-prereqs.yaml`)
4. Deploy Minio: (`kubectl apply -f https://raw.githubusercontent.com/heptio/ark/master/examples/minio/00-minio-deployment.yaml`)
5. Deploy Ark Config: (`kubectl apply -f https://raw.githubusercontent.com/stevesloka/ark-blockstore-rook/master/example/00-ark-config.yaml`)
6. Deploy Ark Server: (`kubectl apply -f https://raw.githubusercontent.com/stevesloka/ark-blockstore-rook/master/example/10-deployment.yaml`)
7. Deploy Sample App: (`kubectl apply -f https://raw.githubusercontent.com/stevesloka/ark-blockstore-rook/master/example/nginx-sample-withpv.yaml`)



