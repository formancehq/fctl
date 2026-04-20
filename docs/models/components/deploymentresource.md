# DeploymentResource


## Fields

| Field                                                 | Type                                                  | Required                                              | Description                                           |
| ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- |
| `ID`                                                  | `string`                                              | :heavy_check_mark:                                    | N/A                                                   |
| `AppID`                                               | `string`                                              | :heavy_check_mark:                                    | N/A                                                   |
| `ManifestID`                                          | `*string`                                             | :heavy_minus_sign:                                    | N/A                                                   |
| `ManifestVersion`                                     | `*int64`                                              | :heavy_minus_sign:                                    | N/A                                                   |
| `HasInlineContent`                                    | `*bool`                                               | :heavy_minus_sign:                                    | N/A                                                   |
| `WorkspaceID`                                         | `string`                                              | :heavy_check_mark:                                    | N/A                                                   |
| `RunID`                                               | `*string`                                             | :heavy_minus_sign:                                    | N/A                                                   |
| `RunStatus`                                           | `string`                                              | :heavy_check_mark:                                    | N/A                                                   |
| `ConfigVersionID`                                     | `*string`                                             | :heavy_minus_sign:                                    | N/A                                                   |
| `CreatedAt`                                           | [time.Time](https://pkg.go.dev/time#Time)             | :heavy_check_mark:                                    | N/A                                                   |
| `UpdatedAt`                                           | [time.Time](https://pkg.go.dev/time#Time)             | :heavy_check_mark:                                    | N/A                                                   |
| `State`                                               | [*components.State](../../models/components/state.md) | :heavy_minus_sign:                                    | N/A                                                   |