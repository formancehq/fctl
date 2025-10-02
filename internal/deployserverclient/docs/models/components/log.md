# Log


## Fields

| Field                                                           | Type                                                            | Required                                                        | Description                                                     |
| --------------------------------------------------------------- | --------------------------------------------------------------- | --------------------------------------------------------------- | --------------------------------------------------------------- |
| `Message`                                                       | *string*                                                        | :heavy_check_mark:                                              | Log message                                                     |
| `Timestamp`                                                     | [time.Time](https://pkg.go.dev/time#Time)                       | :heavy_check_mark:                                              | Timestamp when the log was created                              |
| `Module`                                                        | *string*                                                        | :heavy_check_mark:                                              | Module or component that generated the log                      |
| `Diagnostic`                                                    | [*components.Diagnostic](../../models/components/diagnostic.md) | :heavy_minus_sign:                                              | Detailed log message                                            |