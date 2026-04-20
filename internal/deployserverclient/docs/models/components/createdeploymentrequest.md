# CreateDeploymentRequest


## Fields

| Field                                             | Type                                              | Required                                          | Description                                       |
| ------------------------------------------------- | ------------------------------------------------- | ------------------------------------------------- | ------------------------------------------------- |
| `AppID`                                           | `string`                                          | :heavy_check_mark:                                | ID of the app to deploy to                        |
| `ManifestID`                                      | `*string`                                         | :heavy_minus_sign:                                | Manifest catalog ID (for manifest-reference mode) |
| `ManifestVersion`                                 | `*int64`                                          | :heavy_minus_sign:                                | Manifest version (for manifest-reference mode)    |