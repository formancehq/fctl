# ConfigurationVersion


## Fields

| Field                                                  | Type                                                   | Required                                               | Description                                            |
| ------------------------------------------------------ | ------------------------------------------------------ | ------------------------------------------------------ | ------------------------------------------------------ |
| `ID`                                                   | *string*                                               | :heavy_check_mark:                                     | Unique identifier for the configuration version        |
| `AutoQueueRuns`                                        | *bool*                                                 | :heavy_check_mark:                                     | Auto queue runs when a new version is uploaded         |
| `Error`                                                | *string*                                               | :heavy_check_mark:                                     | Error code if the version is in an error state         |
| `ErrorMessage`                                         | *string*                                               | :heavy_check_mark:                                     | Error message if the version is in an error state      |
| `Status`                                               | [components.Status](../../models/components/status.md) | :heavy_check_mark:                                     | N/A                                                    |