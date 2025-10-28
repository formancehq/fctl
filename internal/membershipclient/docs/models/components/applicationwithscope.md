# ApplicationWithScope


## Fields

| Field                                                  | Type                                                   | Required                                               | Description                                            |
| ------------------------------------------------------ | ------------------------------------------------------ | ------------------------------------------------------ | ------------------------------------------------------ |
| `Name`                                                 | *string*                                               | :heavy_check_mark:                                     | Application name                                       |
| `Description`                                          | **string*                                              | :heavy_minus_sign:                                     | Application description                                |
| `URL`                                                  | *string*                                               | :heavy_check_mark:                                     | Application URL (must be unique)                       |
| `Alias`                                                | *string*                                               | :heavy_check_mark:                                     | Application alias                                      |
| `ID`                                                   | *string*                                               | :heavy_check_mark:                                     | Application ID                                         |
| `CreatedAt`                                            | [time.Time](https://pkg.go.dev/time#Time)              | :heavy_check_mark:                                     | Creation date                                          |
| `UpdatedAt`                                            | [time.Time](https://pkg.go.dev/time#Time)              | :heavy_check_mark:                                     | Last update date                                       |
| `Scopes`                                               | [][components.Scope](../../models/components/scope.md) | :heavy_check_mark:                                     | List of scopes associated with this application        |