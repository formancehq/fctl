# ReadCurrentRunLogsResponse


## Fields

| Field                                                                       | Type                                                                        | Required                                                                    | Description                                                                 |
| --------------------------------------------------------------------------- | --------------------------------------------------------------------------- | --------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `HTTPMeta`                                                                  | [components.HTTPMetadata](../../models/components/httpmetadata.md)          | :heavy_check_mark:                                                          | N/A                                                                         |
| `ReadLogsResponse`                                                          | [*components.ReadLogsResponse](../../models/components/readlogsresponse.md) | :heavy_minus_sign:                                                          | Current run logs retrieved successfully                                     |
| `Error`                                                                     | [*components.Error](../../models/components/error.md)                       | :heavy_minus_sign:                                                          | Error                                                                       |