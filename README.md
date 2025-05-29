# VAI Sidecar

Simple sidecar for accessing Rancher's VAI SQLite database and creating consistent snapshots.

## What it does

This sidecar runs alongside Rancher pods and provides HTTP endpoints to:
- Check health status
- Download VACUUM'd snapshots of the VAI database

## Docker Hub

The image is available at: `brudnak/vai-sidecar:latest`

## Build & Push to Docker Hub
```bash
# Build and push to Docker Hub (brudnak/vai-sidecar:latest)
make push

# Or specify a version
make push VERSION=v1.0.0
```

## Manual Installation via Rancher UI

### 1. Add the Sidecar to Rancher Deployment

1. Login to your Rancher UI
2. Navigate to **Cluster Management** → Select your cluster (usually `local`)
3. Go to **Workloads** → **Deployments**
4. Change namespace to `cattle-system` (dropdown at top)
5. Find the `rancher` deployment and click the **⋮** menu → **Edit Config**

### 2. Configure Shared Storage

First, we need to create a shared volume that both containers can access:

1. Click on the **Pod** tab
2. In the **Volumes** section:
   - If there's an existing `rancher-data` volume, remove it
   - Click **Add Volume** → Select **Empty Dir**
   - Name: `rancher-data`
   - Click **Add Volume**

### 3. Add Volume Mount to Rancher Container

1. Click on the **rancher** container tab
2. Find the **Storage** section
3. Click **Add Mount** and configure:
   - Volume: `rancher-data`
   - Mount Point: `/var/lib/rancher`
   - Read Only: Leave unchecked
   
### 4. Add the Sidecar Container

1. Click **Add Container** and configure:
   - Name: `vai-sidecar`
   - Image: `brudnak/vai-sidecar:latest`
   - Pull Policy: `Always`
   
2. In the same container, go to **Storage**:
   - Click **Add Mount**
   - Volume: `rancher-data`
   - Mount Point: `/var/lib/rancher`
   - Read Only: ✓ Check this box

3. (Optional) Add a health check:
   - Go to **Health Check** section
   - Add **Readiness Probe**:
     - Type: `HTTP`
     - Path: `/health`
     - Port: `8080`

4. Click **Save**

The deployment will restart with the sidecar attached.

## Manual Usage

### 1. Find a Rancher Pod

```bash
# List all rancher pods with the sidecar
kubectl get pods -n cattle-system -l app=rancher

# You should see output like:
# NAME                       READY   STATUS    RESTARTS   AGE
# rancher-594469cd7f-9lhg4   2/2     Running   0          5m
# rancher-594469cd7f-cxclm   2/2     Running   0          5m
# rancher-594469cd7f-f2f8p   2/2     Running   0          5m
```

The `2/2` means both containers (rancher + vai-sidecar) are running.

### 2. Port Forward to Access the Sidecar

```bash
# Pick any pod from above and port-forward
kubectl port-forward -n cattle-system rancher-594469cd7f-9lhg4 8081:8080

# Keep this running in the terminal
# It will show: Forwarding from 127.0.0.1:8081 -> 8080
```

### 3. Use the Sidecar (in a new terminal)

```bash
# Test the health endpoint
curl http://localhost:8081/health
# Should return: OK

# Download a database snapshot
curl http://localhost:8081/snapshot -o vai-snapshot.db

# Check the snapshot
ls -lh vai-snapshot.db
# Should show a file size > 500KB

# Open with SQLite (if you have sqlite3 installed)
sqlite3 vai-snapshot.db ".tables"
# Shows all tables in the database
```

### 4. Get Snapshots from All Pods

To get snapshots from all Rancher pods:

```bash
# Script to get snapshots from all pods
for pod in $(kubectl get pods -n cattle-system -l app=rancher -o jsonpath='{.items[*].metadata.name}'); do
    echo "Getting snapshot from $pod..."
    kubectl port-forward -n cattle-system $pod 8081:8080 &
    PF_PID=$!
    sleep 2
    curl -s http://localhost:8081/snapshot -o snapshot-$pod.db
    kill $PF_PID 2>/dev/null
    echo "Saved snapshot-$pod.db"
done
```

## Troubleshooting

### Pod shows 1/2 containers ready
The sidecar isn't running. Check logs:
```bash
kubectl logs -n cattle-system <pod-name> -c vai-sidecar
```

### Empty or small snapshot file
Make sure both containers share the same volume:
- The volume must be type `emptyDir` not `hostPath`
- Both containers must mount the same volume at `/var/lib/rancher`

### Connection refused on curl
Make sure port-forward is running and use the right port (8081 in examples)

## Endpoints

- `/health` - Health check, returns "OK"
- `/snapshot` - Downloads a VACUUM'd snapshot of the database

## For E2E Testing

This sidecar enables E2E tests to access Rancher's VAI database without kubectl exec commands. Tests can:
1. Port-forward to the sidecar
2. Download snapshots via HTTP
3. Analyze the SQLite database locally

Example test usage:
```go
// Port forward and get snapshot
snapshot, err := downloadSnapshot("rancher-pod-name")
// Analyze with local SQLite
db, err := sql.Open("sqlite3", snapshot)
```