# Scope


## Fields

| Field                                        | Type                                         | Required                                     | Description                                  |
| -------------------------------------------- | -------------------------------------------- | -------------------------------------------- | -------------------------------------------- |
| `ID`                                         | *int64*                                      | :heavy_check_mark:                           | Scope ID                                     |
| `Label`                                      | *string*                                     | :heavy_check_mark:                           | The OAuth2 scope label (e.g., "custom:read") |
| `Description`                                | **string*                                    | :heavy_minus_sign:                           | Scope description                            |
| `ApplicationID`                              | **string*                                    | :heavy_minus_sign:                           | Application ID (null for global scopes)      |
| `Protected`                                  | **bool*                                      | :heavy_minus_sign:                           | Whether the scope is protected               |
| `CreatedAt`                                  | [time.Time](https://pkg.go.dev/time#Time)    | :heavy_check_mark:                           | Creation timestamp                           |
| `UpdatedAt`                                  | [time.Time](https://pkg.go.dev/time#Time)    | :heavy_check_mark:                           | Last update timestamp                        |