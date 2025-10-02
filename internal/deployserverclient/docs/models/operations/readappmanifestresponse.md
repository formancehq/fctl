# ReadAppManifestResponse


## Fields

| Field                                                              | Type                                                               | Required                                                           | Description                                                        |
| ------------------------------------------------------------------ | ------------------------------------------------------------------ | ------------------------------------------------------------------ | ------------------------------------------------------------------ |
| `HTTPMeta`                                                         | [components.HTTPMetadata](../../models/components/httpmetadata.md) | :heavy_check_mark:                                                 | N/A                                                                |
| `ResponseStream`                                                   | *io.ReadCloser*                                                    | :heavy_minus_sign:                                                 | App manifest retrieved successfully                                |
| `Error`                                                            | [*components.Error](../../models/components/error.md)              | :heavy_minus_sign:                                                 | Error                                                              |