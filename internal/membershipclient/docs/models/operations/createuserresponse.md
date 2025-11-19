# CreateUserResponse


## Fields

| Field                                                                           | Type                                                                            | Required                                                                        | Description                                                                     |
| ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| `HTTPMeta`                                                                      | [components.HTTPMetadata](../../models/components/httpmetadata.md)              | :heavy_check_mark:                                                              | N/A                                                                             |
| `CreateUserResponse`                                                            | [*components.CreateUserResponse](../../models/components/createuserresponse.md) | :heavy_minus_sign:                                                              | User created successfully                                                       |
| `Error`                                                                         | [*components.Error](../../models/components/error.md)                           | :heavy_minus_sign:                                                              | Invalid request (missing or invalid email)                                      |