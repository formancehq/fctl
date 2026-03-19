# Module


## Fields

| Field                                                                 | Type                                                                  | Required                                                              | Description                                                           |
| --------------------------------------------------------------------- | --------------------------------------------------------------------- | --------------------------------------------------------------------- | --------------------------------------------------------------------- |
| `Name`                                                                | *string*                                                              | :heavy_check_mark:                                                    | N/A                                                                   |
| `State`                                                               | [components.ModuleState](../../models/components/modulestate.md)      | :heavy_check_mark:                                                    | N/A                                                                   |
| `Status`                                                              | [components.ModuleStatus](../../models/components/modulestatus.md)    | :heavy_check_mark:                                                    | N/A                                                                   |
| `LastStatusUpdate`                                                    | [time.Time](https://pkg.go.dev/time#Time)                             | :heavy_check_mark:                                                    | N/A                                                                   |
| `LastStateUpdate`                                                     | [time.Time](https://pkg.go.dev/time#Time)                             | :heavy_check_mark:                                                    | N/A                                                                   |
| `ClusterStatus`                                                       | [*components.ClusterStatus](../../models/components/clusterstatus.md) | :heavy_minus_sign:                                                    | N/A                                                                   |