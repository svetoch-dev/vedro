# Updates for ArgoCD

When the `../core` library chart is updated, you must run:

`helm dependency update`


inside this chart's directory. This will regenerate the `charts/core-*.tgz` archive. Make sure to **commit the updated `.tgz` file** to version control.

> ⚠️ If this step is skipped, changes made to the `core` chart will **not** be reflected in the `common` Helm chart.

This manual step is required because Helm does **not yet support recursive dependency updates**.
See: [helm/helm#11766](https://github.com/helm/helm/pull/11766)

