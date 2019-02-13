# linode-ddns

Check your external IP address and update Linode DNS with entry.

Useful for home networks on dynamic IPs. Run as either a standalone application or as docker instance or within kubernetes as a cron.

## How it works

After linode-ddns is running it will send a request to ifconfig.co to establish its external IP address and update a specified record in Linode's managed DNS offering.

## Requirements

1. An active account with Linode.
2. A Linode API token scoped down to only allow DNS access.
3. Existing registered domain within Linode.

## Environment Variables

| Variable                 | Explanation                                                |
|--------------------------|------------------------------------------------------------|
| LINODE_TOKEN             | Linode API token scoped to allow DNS read/update actions   |
| DNS_HOSTNAME             | Fully qualified domain name to be updated                  |
| DEBUG                    | Set debug logs (boolean) [Default: false]                  |

## Example Kubernetes deployment

To run linode-ddns on a hourly cron. Update DNS_HOSTNAME and LINODE_TOKEN.

``` kubectl apply -f ./cronjob.yaml ```

```
apiVersion: v1
kind: Secret
metadata:
  name: linode-ddns
type: Opaque
stringData:
  LINODE_TOKEN: XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
---
apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: linode-ddns
spec:
  schedule: "0 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: linode-ddns
            image: jeffvader/linode-ddns
            env:
              - name: LINODE_TOKEN
                valueFrom:
                  secretKeyRef:
                    name: linode-ddns
                    key: LINODE_TOKEN
              - name: DNS_HOSTNAME
                value: hostname.example.com
              - name: DEBUG
                value: "true"
            command: ['/linode-ddns']
          restartPolicy: OnFailure
```