# ReadAppCurrentStateVersionResponse


## Fields

| Field                                                                         | Type                                                                          | Required                                                                      | Description                                                                   |
| ----------------------------------------------------------------------------- | ----------------------------------------------------------------------------- | ----------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| `HTTPMeta`                                                                    | [components.HTTPMetadata](../../models/components/httpmetadata.md)            | :heavy_check_mark:                                                            | N/A                                                                           |
| `ReadStateResponse`                                                           | [*components.ReadStateResponse](../../models/components/readstateresponse.md) | :heavy_minus_sign:                                                            | Current state version retrieved successfully                                  |
| `Error`                                                                       | [*components.Error](../../models/components/error.md)                         | :heavy_minus_sign:                                                            | Error                                                                         |