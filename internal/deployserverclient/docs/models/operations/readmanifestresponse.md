# ReadManifestResponse


## Fields

| Field                                                                       | Type                                                                        | Required                                                                    | Description                                                                 |
| --------------------------------------------------------------------------- | --------------------------------------------------------------------------- | --------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| `HTTPMeta`                                                                  | [components.HTTPMetadata](../../models/components/httpmetadata.md)          | :heavy_check_mark:                                                          | N/A                                                                         |
| `ManifestResponse`                                                          | [*components.ManifestResponse](../../models/components/manifestresponse.md) | :heavy_minus_sign:                                                          | Manifest retrieved successfully                                             |
| `ResponseStream`                                                            | `io.ReadCloser`                                                             | :heavy_minus_sign:                                                          | Manifest retrieved successfully                                             |
| `Error`                                                                     | [*components.Error](../../models/components/error.md)                       | :heavy_minus_sign:                                                          | Error                                                                       |