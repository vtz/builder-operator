# Examples

- `hello-world-shared-folder.yaml`: local-friendly hello world C++ pipeline writing artifacts to shared folder.
- `git-fetch.yaml`: git clone fetch stage with token-based secret reference.
- `artifactory-deploy.yaml`: demonstrates artifactory destination config and secret reference.

Apply any example:

```bash
kubectl apply -f docs/examples/<example>.yaml
```
